import "@typespec/http";
import "@typespec/openapi";
import "../common/error.tsp";
import "../shared/index.tsp";
import "./namespaces.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Namespaces;

interface NamespacesOperations {
  /**
   * List the namespaces available on this deployment. The default namespace is
   * always included in the response.
   */
  @get
  @operationId("list-namespaces")
  @summary("List namespaces")
  @extension(Shared.UnstableExtension, true)
  list(): NamespaceList | Common.ErrorResponses;
}
