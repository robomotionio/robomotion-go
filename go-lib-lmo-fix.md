# Go SDK LMO Fix Summary

## Problem 1: `ctx.Get()` / `ctx.GetString()` / etc. did not resolve BlobRefs

### What happened

The Go SDK's `message.Context` methods (`Get`, `GetString`, `GetBool`, `GetInt`,
`GetFloat`) all delegated to an internal `get()` method that called
`gjson.GetBytes(msg.data, path)` directly — raw JSON access with no LMO awareness.

When LMO was active and the robot packed a message field into a BlobRef, any Go
package calling `ctx.GetString("field")` received the serialised BlobRef JSON
object (`{"__ref":"xxh3:...","__magic":20260301,...}`) instead of the actual value.

The only way to get resolved data was `ctx.GetRaw(runtime.WithUnpack())`, which
resolved **all** fields eagerly. Individual field access had no resolution path.

Both the .NET SDK (`Message.Get<T>()` → `LMO.Resolve`) and the Python SDK
(`Message.get()` → `lmo_resolve`) already resolved BlobRefs transparently in their
Get methods. The Go SDK was the only one that did not.

### Fix

Added a `Resolve` function variable to the `message` package (same pattern as the
existing `GetRaw` / `SetRaw` wiring). The `get()` method now calls `Resolve` first;
if it returns a valid result, that is used. If `Resolve` is nil (runtime not
imported) or returns an error, the method falls back to plain `gjson.GetBytes`.

`runtime/message.go` sets `message.Resolve` during `init()` to call
`LMOResolveSubtree` (see Problem 2 below).

### Files changed

- `message/context.go` — added `Resolve` var, updated `get()`
- `runtime/message.go` — wired `LMOResolveSubtree` into `message.Resolve`

---

## Problem 2: Lazy resolve did not handle nested BlobRefs inside objects

### What happened

The LMO packing algorithm recurses into JSON objects. When a parent object is
larger than 4 KB, the packer inspects each child:

1. Children >= 4 KB are individually extracted as BlobRefs.
2. If at least one child was extracted, the modified parent (with BlobRef children)
   is kept as a regular JSON object — it is **not** turned into a BlobRef itself.

Example after packing:

```json
{
  "executionData": {
    "col1": "short",
    "col2": {"__ref":"xxh3:...","__magic":20260301,"__type":"string","__len":5387},
    "col3": 123
  }
}
```

`executionData` is a plain object. `col2` inside it is a BlobRef.

`LMOResolve(data, "executionData")` found the object, checked `IsBlobRef` — false
(it is a regular object) — and returned it as-is. The BlobRef nested inside `col2`
was never resolved.

Any node that reads the parent object as a whole (e.g. SQLite Insert reading a
table row, or a Function node working with a complex object) received raw BlobRef
marker dicts instead of actual values.

Accessing the leaf directly (`LMOResolve(data, "executionData.col2")`) worked
correctly because the path walker found the BlobRef at that exact location and
resolved it. This is why writing the same string to a txt file (which accesses the
specific field) worked while the SQLite Insert (which reads the whole object) did
not.

### Fix

Added `LMOResolveSubtree` in `runtime/lmo_rt.go`:

1. Calls `LMOResolve` to get the field (lazy, single-level).
2. If the result is a JSON object or array, calls `lmoStore.ResolveAll` on it to
   recursively resolve every nested BlobRef.
3. For scalars or when the store is nil: returns immediately with zero overhead.

This is the same pattern the deskbot already used internally for built-in nodes
(e.g. `LMOResolveSubtree` in the deskbot's own `lmo_rt.go` for the HTTP Request
node body extraction).

### Files changed

- `runtime/lmo_rt.go` — added `LMOResolveSubtree`
- `runtime/message.go` — `message.Resolve` wired to `LMOResolveSubtree` (not `LMOResolve`)

### Version

Bumped to **v1.15.0** (git tag).
