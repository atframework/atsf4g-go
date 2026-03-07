# Code Generation (proto / config / RPC)

This repo generates protobuf descriptors, RPC task handlers, and config/data artifacts.

## Custom Protobuf Plugin (protoc-gen-mutable)

We use a custom protobuf plugin (`protoc-gen-mutable`) that generates additional helper code beyond the standard protobuf-go output.
The generated code lives in `*_mutable.pb.go` files alongside the standard `*.pb.go` files.

### Generated Features

1. **Mutable Methods for Messages**: For each message field, generates `Mutable<FieldName>()` methods that create-if-nil and return the field:
   ```go
   // Standard getter (returns nil if not set)
   msg.GetHead()
   // Mutable getter (creates if nil, always returns non-nil)
   msg.MutableHead()
   ```

2. **Clone and Merge Methods**: For each message:
   ```go
   clone := msg.Clone()           // Deep copy
   msg.Merge(other)               // Merge other into msg
   ```

3. **Oneof Case Enum Constants**: For each oneof field, generates `MessageBody_EnMessageTypeID` enum:
   ```go
   // Get which oneof case is set
   caseID := body.GetMessageTypeOneofCase()
   // Compare against constants
   if caseID == protocol.MessageBody_EnMessageTypeID_DataTransformReq { ... }
   ```

4. **Oneof Mutable Methods**: Access oneof fields with create-if-nil semantics:
   ```go
   // Sets the oneof to DataTransformReq if not already, returns mutable inner message
   forwardData := body.MutableDataTransformReq()
   ```

5. **Readonly Wrappers**: Immutable wrapper types for safe read-only access:
   ```go
   ro := msg.ToReadonly()  // Returns *Readonly_MessageHead
   val := ro.GetVersion()  // Safe read-only access
   ```

6. **Reflect Type Helpers**: Get reflect.Type for messages:
   ```go
   reflectType := msg.GetReflectType()
   ```

7. **Slog Integration**: Messages implement `slog.LogValuer` for structured logging:
   ```go
   slog.Info("received message", "msg", msg)  // Logs readable proto text
   ```

### Usage Guidelines

- **Prefer Mutable methods** over manual nil-check + create patterns
- **Use oneof enum constants** (e.g., `MessageBody_EnMessageTypeID_DataTransformReq`) instead of hardcoded field numbers
- **Use Clone()** for safe copies, avoid sharing proto message pointers across goroutines
- **Reference generated constants** from protocol package instead of copying literal values

## Where templates and rules live

- Mako templates:
  - `src/template/handle_cs_rpc.go.mako`
  - `src/template/task_action_cs_rpc.go.mako`
- Per-module rule templates (source of truth):
  - `src/lobbysvr/generate-for-pb.atfw.yaml.tpl`
  - `src/robot/generate-for-pb.atfw.yaml.tpl`
  - (and other `generate-for-pb.atfw.*.yml/.yaml.tpl` files)
- Generated merged rule file (output):
  - `build/_gen/generate-for-pb.yaml`

## Typical workflows (gotask)

- Generate protocol descriptors (component level):
  - `task component:generate-protocol`
- Generate config outputs (component level):
  - `task component:generate-config`
- Generate + build per service:
  - `task build:lobbysvr`
  - `task build:robot`

## Changing/adding RPC

1. Update the relevant `.proto` definitions.
2. Update the corresponding generation rule template under `src/*/generate-for-pb.atfw.yaml.tpl` if needed.
3. Run `task build:<module>` (or the module generation tasks) to regenerate.

## Notes

- Treat `build/_gen/generate-for-pb.yaml` as generated output, not editable source.
- Tool/bootstrap parameters are driven by `/.taskfiles/build-tools.env`.

