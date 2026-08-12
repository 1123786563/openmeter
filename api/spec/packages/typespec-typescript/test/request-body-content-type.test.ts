import { beforeAll, describe, expect, it } from 'vitest'
import { EmitterTester } from './emit.js'

const FIXTURE = `
import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Callbacks {
  model Widget {
    id: string;
  }

  interface CallbackOperations {
    @post
    @route("/json-string")
    @operationId("send-json-string")
    sendJsonString(
      @header contentType: "application/json",
      @body body: string,
    ): void;

    @post
    @route("/form-string")
    @operationId("send-form-string")
    sendFormString(
      @header contentType: "application/x-www-form-urlencoded",
      @body body: string,
    ): void;

    @post
    @route("/normal-json")
    @operationId("send-normal-json")
    sendNormalJson(@body body: Widget): void;
  }
}

@service(#{ title: "Test API" })
namespace Api {
  @route("/callbacks")
  interface CallbackEndpoints extends Callbacks.CallbackOperations {}
}
`

describe('request body content types', () => {
  let funcs: string

  beforeAll(async () => {
    const [result, diagnostics] =
      await EmitterTester.compileAndDiagnose(FIXTURE)
    expect(
      diagnostics.filter((d) => d.severity === 'error'),
      'fixture must compile without errors',
    ).toEqual([])
    funcs = result.outputs['src/funcs/callbacks.ts']!
  })

  function operation(name: string): string {
    const start = funcs.indexOf(`export function ${name}(`)
    expect(start, `expected generated function ${name}`).toBeGreaterThan(-1)
    const next = funcs.indexOf('\nexport function ', start + 1)
    return funcs.slice(start, next === -1 ? undefined : next)
  }

  it('sends an application/json string as the raw request body', () => {
    const generated = operation('sendJsonString')

    expect(generated).toContain(
      "headers.set('content-type', 'application/json')",
    )
    expect(generated).toContain(".post('callbacks/json-string', {")
    expect(generated).toContain('body,')
    expect(generated).not.toContain('json: body')
  })

  it('sends a form-urlencoded string as the raw request body', () => {
    const generated = operation('sendFormString')

    expect(generated).toContain(
      "headers.set('content-type', 'application/x-www-form-urlencoded')",
    )
    expect(generated).toContain(".post('callbacks/form-string', {")
    expect(generated).toContain('body,')
    expect(generated).not.toContain('json: body')
  })

  it('keeps normal JSON object operations on the json request option', () => {
    const generated = operation('sendNormalJson')

    expect(generated).toContain(
      ".post('callbacks/normal-json', { ...options, json: body })",
    )
    expect(generated).not.toContain("headers.set('content-type'")
  })
})
