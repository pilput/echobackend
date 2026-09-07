# API Documentation - echobackend

The HTTP API is described by an OpenAPI 3.1 document, split into small per-module files so
a reader — human or AI agent — only has to open the module it cares about instead of one
giant file.

```
docs/api/
├── openapi.yaml        # root: info, servers, security scheme, and the paths index
├── parameters.yaml      # shared query/path parameters (limit, offset, IDs, ...)
├── responses.yaml       # shared error responses (400/401/403/404/409/422/429/500)
├── paths/                # one file per module — the actual endpoint definitions
│   ├── auth.yaml
│   ├── user.yaml         # includes follow/unfollow
│   ├── post.yaml         # includes comments, views, likes
│   ├── tag.yaml
│   ├── chat.yaml
│   ├── bookmark.yaml
│   ├── notification.yaml
│   ├── holding.yaml      # includes holding-types and the corporate-actions calendar
│   ├── exchange-rate.yaml
│   └── report.yaml
└── schemas/              # one file per module — request/response schemas
    ├── envelope.yaml     # SuccessEnvelope, ErrorEnvelope, PaginationMeta, ...
    ├── auth.yaml
    ├── user.yaml
    ├── post.yaml
    ├── tag.yaml
    ├── chat.yaml
    ├── bookmark.yaml
    ├── notification.yaml
    ├── holding.yaml
    ├── exchange-rate.yaml
    └── report.yaml
```

`openapi.yaml` never inlines an operation — every path entry is a one-line `$ref` into
`paths/<module>.yaml`. To look up one endpoint: find its line in `openapi.yaml`, open the
file it points to. A module's path file and schema file rarely exceed a few hundred lines
each (the biggest, `paths/post.yaml`, is ~800), against ~4,600 lines combined if this were
one file — so reading just what's needed costs a fraction of the tokens or scrolling.

This is plain OpenAPI multi-file referencing (`$ref` to external files) — every mainstream
tool resolves it natively (Redocly, Swagger UI, Postman, openapi-generator, editor
extensions). Nothing here is bespoke.

## Viewing it

```bash
# Interactive docs in the browser (resolves all the split files automatically)
npx @redocly/cli preview-docs docs/api/openapi.yaml

# One-off HTML build
npx @redocly/cli build-docs docs/api/openapi.yaml -o docs/api/index.html
```

Any OpenAPI viewer works — Swagger UI, Scalar, Bruno, Insomnia, Postman (Import → File),
or the editor extensions for VS Code / JetBrains. Tools that require a single file (some
older codegens) need a bundle first — see below.

## Validating it

```bash
npx @redocly/cli lint docs/api/openapi.yaml
```

Lints the fully resolved document across all the split files. Expected to pass with
**0 errors**. The remaining warnings are inherent to the service and intentionally left
in place:

- `no-ambiguous-paths` — Echo registers static segments alongside parameterised ones
  (`/api/posts/random` next to `/api/posts/:id`); Echo resolves the static route first.
- `operation-4xx-response` — some public read endpoints genuinely only fail with a 500.
- `operation-2xx-response` — the GitHub OAuth endpoints answer with a 307 redirect only.
- `no-server-example.com` — the first server entry is `localhost`, for local development.
- `info-license` — the repository carries no licence file.

## Bundling into one file

Needed for tools that don't support multi-file `$ref` (some older codegens), or to hand
someone a single self-contained file:

```bash
npx @redocly/cli bundle docs/api/openapi.yaml -o docs/api/openapi.bundled.yaml
```

Don't hand-edit the bundled output — it's generated; edit the split source files instead
and re-bundle.

## Generating clients

```bash
npx @openapitools/openapi-generator-cli generate \
  -i docs/api/openapi.yaml -g typescript-fetch -o ./client
```

## Keeping it current

The spec is written by hand — there is no code generation step, and no build check enforces
it. When a route, DTO field, or status code changes in `internal/`:

1. Edit the relevant `paths/<module>.yaml` and/or `schemas/<module>.yaml`.
2. Adding a brand-new endpoint also means adding its one-line `$ref` entry to the `paths:`
   map in the root `openapi.yaml`.
3. Re-run `npx @redocly/cli lint docs/api/openapi.yaml` — must stay at 0 errors — in the
   same commit.
