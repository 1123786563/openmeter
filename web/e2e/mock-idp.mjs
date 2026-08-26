#!/usr/bin/env node
/**
 * Minimal OIDC provider (mock IdP) for the Playwright smoke tests.
 *
 * Implements just enough of the authorization code + PKCE flow for
 * `oidc-client-ts` to complete a real login round-trip against the admin
 * SPA without a Casdoor instance. Pure Node (node:http + node:crypto), no
 * third-party dependencies.
 *
 * Endpoints (all on 127.0.0.1:9999 unless MOCK_IDP_PORT is set):
 *   GET  /.well-known/openid-configuration
 *   GET  /authorize   -> 302 redirect_uri?code=...&state=...
 *   POST /token       -> access_token + RS256 id_token
 *   GET  /jwks.json
 *
 * Nonce handling: the authorization request carries the nonce (and the
 * client only returns it inside the id_token), while the browser comes
 * back to the redirect_uri with code + state only. The IdP therefore
 * stores nonce keyed by the issued code at /authorize time and injects it
 * back into the id_token claims at /token time, so the SPA's default
 * nonce validation passes.
 *
 * An RSA keypair is generated at startup (kid: mock-idp-key-1) and exposed
 * via /jwks.json so the client can verify the RS256 signature.
 */
import { createServer } from 'node:http'
import { createSign, generateKeyPairSync, randomUUID } from 'node:crypto'

const HOST = '127.0.0.1'
const PORT = Number(process.env.MOCK_IDP_PORT ?? 9999)
const ISSUER = `http://${HOST}:${PORT}`
const CLIENT_ID = process.env.MOCK_IDP_CLIENT_ID ?? 'openmeter-admin-e2e'
const KID = 'mock-idp-key-1'
const ACCESS_TOKEN_TTL_SECONDS = 3600

const { publicKey, privateKey } = generateKeyPairSync('rsa', {
  modulusLength: 2048,
})
// The JWK export already encodes n/e as base64url, exactly what /jwks.json
// needs.
const publicJwk = publicKey.export({ format: 'jwk' })

/** code -> { nonce, clientId } map filled at /authorize, drained at /token. */
const codes = new Map()

function base64url(input) {
  return Buffer.from(input).toString('base64url')
}

/** Sign a compact RS256 JWT with the startup key. */
function signIdToken(payload) {
  const header = { alg: 'RS256', kid: KID, typ: 'JWT' }
  const unsigned = `${base64url(JSON.stringify(header))}.${base64url(
    JSON.stringify(payload)
  )}`
  const signature = createSign('RSA-SHA256').update(unsigned).sign(privateKey)
  return `${unsigned}.${signature.toString('base64url')}`
}

function sendJson(res, status, body) {
  const data = JSON.stringify(body)
  res.writeHead(status, {
    'Content-Type': 'application/json',
    'Content-Length': Buffer.byteLength(data),
    // oidc-client-ts fetches discovery/JWKS and exchanges the code from the
    // SPA origin, so every response must be CORS-readable.
    'Access-Control-Allow-Origin': '*',
  })
  res.end(data)
}

function sendError(res, status, error, description) {
  sendJson(res, status, { error, error_description: description })
}

function handleDiscovery(res) {
  sendJson(res, 200, {
    issuer: ISSUER,
    authorization_endpoint: `${ISSUER}/authorize`,
    token_endpoint: `${ISSUER}/token`,
    jwks_uri: `${ISSUER}/jwks.json`,
    response_types_supported: ['code'],
    subject_types_supported: ['public'],
    id_token_signing_alg_values_supported: ['RS256'],
    code_challenge_methods_supported: ['S256'],
    token_endpoint_auth_methods_supported: ['none'],
    scopes_supported: ['openid', 'profile'],
  })
}

function handleAuthorize(req, res) {
  const query = new URL(req.url, ISSUER).searchParams

  for (const param of ['client_id', 'redirect_uri', 'state', 'code_challenge']) {
    if (!query.get(param)) {
      sendError(res, 400, 'invalid_request', `missing ${param} parameter`)
      return
    }
  }

  const code = randomUUID()
  codes.set(code, {
    nonce: query.get('nonce'),
    clientId: query.get('client_id'),
  })

  const redirectUri = query.get('redirect_uri')
  const separator = redirectUri.includes('?') ? '&' : '?'
  const location = `${redirectUri}${separator}code=${encodeURIComponent(code)}&state=${encodeURIComponent(query.get('state'))}`
  console.log(`[mock-idp] authorize -> code for ${redirectUri}`)
  res.writeHead(302, { Location: location })
  res.end()
}

async function readBody(req) {
  const chunks = []
  for await (const chunk of req) chunks.push(chunk)
  return Buffer.concat(chunks).toString('utf8')
}

async function handleToken(req, res) {
  const form = new URLSearchParams(await readBody(req))

  if (form.get('grant_type') !== 'authorization_code') {
    sendError(res, 400, 'unsupported_grant_type', 'only authorization_code is supported')
    return
  }

  const issued = codes.get(form.get('code'))
  codes.delete(form.get('code'))
  if (!issued) {
    sendError(res, 400, 'invalid_grant', 'unknown or already used code')
    return
  }

  const now = Math.floor(Date.now() / 1000)
  const claims = {
    iss: ISSUER,
    aud: issued.clientId,
    sub: 'e2e-user-1',
    iat: now,
    exp: now + ACCESS_TOKEN_TTL_SECONDS,
    name: 'E2E Test User',
    preferred_username: 'e2e-user',
    email: 'e2e-user@example.com',
  }
  // oidc-client-ts validates the nonce when present in the transaction;
  // inject it only when the authorize request carried one.
  if (issued.nonce) claims.nonce = issued.nonce

  console.log('[mock-idp] token -> id_token issued')
  sendJson(res, 200, {
    access_token: 'mock-access-token',
    id_token: signIdToken(claims),
    token_type: 'Bearer',
    expires_in: ACCESS_TOKEN_TTL_SECONDS,
    scope: 'openid profile',
  })
}

function handleJwks(res) {
  sendJson(res, 200, {
    keys: [
      {
        kty: 'RSA',
        use: 'sig',
        alg: 'RS256',
        kid: KID,
        n: publicJwk.n,
        e: publicJwk.e,
      },
    ],
  })
}

const server = createServer((req, res) => {
  // Allow preflights in case a client future needs them.
  if (req.method === 'OPTIONS') {
    res.writeHead(204, {
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',
      'Access-Control-Allow-Headers': 'Content-Type, Authorization',
    })
    res.end()
    return
  }

  const path = new URL(req.url, ISSUER).pathname

  if (req.method === 'GET' && path === '/.well-known/openid-configuration') {
    handleDiscovery(res)
  } else if (req.method === 'GET' && path === '/authorize') {
    handleAuthorize(req, res)
  } else if (req.method === 'POST' && path === '/token') {
    void handleToken(req, res)
  } else if (req.method === 'GET' && path === '/jwks.json') {
    handleJwks(res)
  } else {
    sendError(res, 404, 'not_found', `no handler for ${req.method} ${path}`)
  }
})

server.listen(PORT, HOST, () => {
  console.log(`[mock-idp] listening on ${ISSUER} (client_id: ${CLIENT_ID})`)
})
