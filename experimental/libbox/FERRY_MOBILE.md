# Optional Ferry runtime support

The default build keeps the upstream include registries and existing platform
interface. No factory is called at server construction. A Go platform adapter
may additionally implement:

```go
NewServiceContext(parent context.Context, forCheck bool) (context.Context, error)
```

This is detected using an unexported, optional interface assertion; it does not
add context.Context to the Java/Swift PlatformInterface contract. The daemon
calls it after deriving an isolated, cancellable context for each run/reload,
before decoding options. Return fresh registries/run state derived from parent,
preserving platform, file-manager and other host services. In particular,
box.Context preserves existing registries; callers replacing default registries
must replace both the options and runtime registry entries explicitly.

Errors and nil contexts fail the operation without default-registry fallback.
forCheck is true for the existing StartedService.CheckConfig Go method and the
daemon gRPC check path; the embedded method signature remains unchanged.
Checks must not publish an active generation, open a TUN, or start background
work. The libbox bridge replaces the real platform interface with the existing
validation stub for these checks; box.New/Close validates constructors without
Start or mutation of the live platform's network manager. Check generations
are canceled on both success and error. Factories remain responsible for not
performing their own side effects and for respecting lifecycle cancellation.

With ferry_mobile, no default include registry is linked by libbox. A factory is
mandatory before NewCommandServer performs setup work. Standalone CheckConfig,
FormatConfig, GenerateConfigSchema and HasTunInbound have no factory argument
and explicitly fail; use StartedService.CheckConfig for custom configurations.
This tag targets the caller-managed libbox library, not the stock sing-box CLI.

Native shell/pipe sessions and CreateZipArchive retain their gomobile signatures
but explicitly report that this build does not support them. Archive export is
disabled for both encrypt values: an encrypted export must never silently become
plaintext. Crash-output redirection, configuration snapshots and the non-age
report helpers remain available, as do ordinary core log streams. Existing
PlatformInterface methods remain unchanged; Ferry's business facade must expose
only its fixed operations rather than forwarding libbox's full API.

The tag does not provide a protocol registry. Ferry must supply its own unchanged
whitelist, custom routing outbound, managed URLTest/probe engine, redacted mixed
inbound, per-profile options/cache identity, and platform-local DNS constructor.
The parent file-manager directories come from existing libbox SetupOptions.
The Go facade/platform adapter and gomobile/Android integration are separate work.
