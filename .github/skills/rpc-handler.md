# RPC Task Action Pattern

Many RPC handlers are generated and follow a consistent “task action” flow.

## Checklist

1. Extract user/session from context (nil-check).
2. Fetch the domain manager via `data.UserGetModuleManager[T](user)` (nil-check).
3. Validate request params; return a proper error code on invalid input.
4. Call the business method; capture `cd.RpcResult`.
5. On error:
   - `t.SetResponseCode(result.GetResponseCode())`
   - `return result.GetStandardError()`
6. On success: populate response body if needed; `return nil`.

## Error handling

- Prefer `cd.CreateRpcResultOk()` for success.
- Prefer `cd.CreateRpcResultError(err, code)` for client-facing error codes.

