import { describe, expect, it } from 'vitest'
import { OpenMeter } from '../src/index.js'

interface CapturedRequest {
  contentType: string | null
  body: string
}

function captureRequest(response: Response): {
  sdk: OpenMeter
  captured: () => CapturedRequest
} {
  let captured: CapturedRequest | undefined
  const fetch: typeof globalThis.fetch = async (input, init) => {
    const request =
      input instanceof Request && init === undefined
        ? input
        : new Request(input, init)
    captured = {
      contentType: request.headers.get('content-type'),
      body: await request.clone().text(),
    }
    return response
  }

  return {
    sdk: new OpenMeter({
      baseUrl: 'https://example.test',
      apiKey: 'test-key',
      fetch,
    }),
    captured: () => {
      expect(captured, 'expected the SDK to call fetch').toBeDefined()
      return captured!
    },
  }
}

describe('commerce callback request transport', () => {
  it('sends the WeChat payment callback as raw application/json', async () => {
    const transport = captureRequest(new Response(null, { status: 204 }))
    const payload = '{"id":"PAYMENT.EVENT","resource":{"ciphertext":"payment"}}'

    await transport.sdk.commerce.wechatPaymentCallback(payload)

    expect(transport.captured()).toEqual({
      contentType: 'application/json',
      body: payload,
    })
  })

  it('sends the WeChat refund callback as raw application/json', async () => {
    const transport = captureRequest(new Response(null, { status: 204 }))
    const payload = '{"id":"REFUND.EVENT","resource":{"ciphertext":"refund"}}'

    await transport.sdk.commerce.wechatRefundCallback(payload)

    expect(transport.captured()).toEqual({
      contentType: 'application/json',
      body: payload,
    })
  })

  it('sends the Alipay callback as raw form-urlencoded data', async () => {
    const transport = captureRequest(
      new Response('success', {
        headers: { 'Content-Type': 'text/plain' },
      }),
    )
    const payload = 'trade_status=TRADE_SUCCESS&trade_no=202608120001'

    await transport.sdk.commerce.alipayPaymentCallback(payload)

    expect(transport.captured()).toEqual({
      contentType: 'application/x-www-form-urlencoded',
      body: payload,
    })
  })
})
