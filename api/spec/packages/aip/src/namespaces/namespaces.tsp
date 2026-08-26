import "@typespec/openapi";

using TypeSpec.OpenAPI;

namespace Namespaces;

/**
 * The namespaces available on this deployment.
 *
 * The list is sourced from the server configuration: the default namespace plus
 * the configured namespace allowlist, deduplicated and sorted. When request-level
 * namespace selection is disabled (empty allowlist), only the default namespace is
 * listed.
 */
@friendlyName("NamespaceList")
model NamespaceList {
  /**
   * The namespace used for requests that do not select one explicitly.
   */
  @summary("Default namespace")
  default: string;

  /**
   * All selectable namespaces, sorted alphabetically. Always includes the default
   * namespace.
   */
  @summary("Namespaces")
  namespaces: string[];
}
