# 🏛️ Pogonian Universal Architecture Specification

Pogonian is a **universal, cross-platform application framework and platform** built around a single unifying principle:

> **ONE APPLICATION → ONE COMMON COMPILER → PLATFORM-SPECIFIC NATIVE OUTPUT**

Applications are authored once against a platform-independent application model, compiled into a unified **Pogonian Intermediate Representation (IR)**, semantically validated, and lowered into genuinely native implementations for each target platform without relying on WebViews or Electron-style containers.

---

## 🏗️ Universal Compiler & Lowering Pipeline

```text
                        Pogonian Application Source
        (Pages, Layouts, Components, Entities, Services, State, Capabilities)
                                     │
                                     ▼
                  ┌─────────────────────────────────────┐
                  │          Pogonian Compiler          │
                  │  - Source & AST Parser              │
                  │  - Semantic Analyzer                │
                  │  - Capability & Boundary Validator  │
                  │  - Architecture Optimizer           │
                  └──────────────────┬──────────────────┘
                                     │
                                     ▼
                  ┌─────────────────────────────────────┐
                  │    Universal Pogonian IR (v2.0)     │
                  │  - Platform-Agnostic UI AST         │
                  │  - Portable Application State       │
                  │  - Event & Action Schemas           │
                  │  - Device Capability Matrix         │
                  │  - Data Models & Service Contracts  │
                  └──────────────────┬──────────────────┘
                                     │
                   Platform Lowering & Code Generation
           ┌───────────┬────────────┼───────────┬───────────┬───────────┐
           ▼           ▼            ▼           ▼           ▼           ▼
      ┌─────────┐ ┌─────────┐  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐
      │   Web   │ │ Android │  │   iOS   │ │  Linux  │ │ Windows │ │Android TV│
      │ Lowerer │ │ Lowerer │  │ Lowerer │ │ Lowerer │ │ Lowerer │ │ Lowerer │
      └────┬────┘ └────┬────┘  └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘
           │           │            │           │           │           │
           ▼           ▼            ▼           ▼           ▼           ▼
      HTML+HTMX+  Native APK/  Native App/ Native ELF/ Native EXE/ Native TV/
      runtime.js  Compose      SwiftUI     GTK4/C-ABI  WinUI/C-ABI 10-Foot UI
      (Server-Go) Runtime      Runtime     Runtime     Runtime     D-Pad Focus
```

---

## 📱 Platform-Specific Native Targets (No WebViews)

Pogonian generates and compiles into genuinely native platform applications without WebViews:

| Target Platform | Lowering Mechanism | Native UI & Input Technology |
| :--- | :--- | :--- |
| **Web** | `WebLowerer` | Server-driven HTML5, soft-navigation `runtime.js`, template compilation |
| **Android** | `AndroidLowerer` | Android Jetpack Compose declarative UI, Material3, Kotlin background HTTP dispatch, Biometrics & Permissions |
| **iOS** | `IOSLowerer` | Native SwiftUI views, `@ObservedObject` reactive state, asynchronous `URLSession` service calls, Biometrics |
| **Linux Desktop** | `DesktopLowerer` | Standalone Go native desktop GUI, native window lifecycle, X11/desktop canvas renderer, event loop, `go build` |
| **Windows Desktop**| `DesktopLowerer` | Standalone Go native desktop GUI, Win32 message pump hooks, reactive `AppState`, native event loop, `go build` |
| **macOS Desktop** | `DesktopLowerer` | Standalone Go native desktop GUI, Cocoa run loop hooks, reactive `AppState`, native event loop, `go build` |
| **Android TV / Google TV** | `TVLowerer` | 10-foot spatial navigation, D-pad key event handling (`KEYCODE_DPAD_*`), Compose TV, Leanback project scaffold |

---

## ⚡ Universal Pogonian Intermediate Representation (IR v2.0)

Located in [`pkg/ir/ir.go`](file:///home/vraj/Desktop/Vruket/Pogonian/Pogonian/pkg/ir/ir.go), the universal IR encapsulates complete application semantics independently of rendering technology:

1. **Platform-Agnostic UI Trees (`UIElementIR`)**:
   - Normalized semantic primitives: `StackVertical`, `StackHorizontal`, `Container`, `Heading`, `Text`, `Button`, `Input`, `Form`, `List`, `Image`, `Card`, `Navigation`, `Grid`, `Modal`, `Spacer`, `Divider`, `Icon`.
   - Layout tokens: Alignment, padding, spacing, flex direction, and responsive constraints.
2. **Application State Schema (`StateIR`)**:
   - Typed reactive variables: `string`, `int`, `bool`, `list`, `map`, `entity:<Name>`.
   - Explicit Scopes: `local` (component), `route` (page), `session` (user auth), `persistent` (storage), `sync` (cross-device).
3. **Event & Action Model (`EventIR`, `ActionIR`)**:
   - Input events: `click`, `change`, `submit`, `navigate`, `mount`, `focus`, `dpad_up`, `dpad_down`, `dpad_select`, `back_press`.
   - Actions: `mutate_state`, `navigate_to`, `call_service` (real backend network dispatch), `request_capability`, `transfer_state`.
4. **Device Capability Requirements (`CapabilityIR`)**:
   - Hardware and platform capabilities with fallback policies and permission manifests.
5. **Authoritative Mount Architecture (`MountIR`)**:
   - Architecture-driven resource bindings: `Resource ➔ UI Route ➔ Handler ➔ Service ➔ Repository ➔ Entity ➔ Table`.

---

## 🔗 Authoritative Mount System (UI → Backend → Data)

The **Mount System** bridges application resources, discovered filesystem routes, backend business logic, and database persistence:

```text
Application Resource (e.g. "products")
             │
             ├──► AST Graph Traversal: Handler ➔ Service ➔ Repository ➔ Entity ➔ Table
             │
             ▼
      Authoritative MountIR
  (Resource: "products", Entity: "Product", Table: "products", PrimaryKey: "id")
             │
             ├──► UI Routes (/products, /products/[id]) bound via ParamBinding
             ├──► Backend Handlers & Services for Business Logic
             └──► Data Runtime Engine (SQLite WAL Persistence)
```

- **Architecture-Driven Resolution**: Mounts are derived from the actual Go AST symbol graph rather than route-name table guessing.
- **Deep Compile-Time Verification**:
  - **Route Parameter ↕ Entity Primary Key**: Ensures dynamic route parameters (e.g. `[id]`) correspond directly to verified primary key or entity fields.
  - **Call Graph Integrity**: Enforces that Handlers reference existing Services, Services reference existing Repositories, and Repositories query registered database tables.
- **Target Source Escaping**: All emitted code across Kotlin, Swift, Go, and XML undergoes strict target-specific escaping to guarantee syntactically valid builds.
- **Configurable Native Packages**: Package IDs and organizations are resolved via `pogonian.yaml` (`cfg.GetPackageID()`).

---

## 💾 Pogonian Data Runtime (Storage Abstraction)

Implemented in [`pkg/dataruntime/runtime.go`](file:///home/vraj/Desktop/Vruket/Pogonian/Pogonian/pkg/dataruntime/runtime.go):

Rather than forcing developers to configure database connection strings or pick database dialects, Pogonian provides a clean storage abstraction above physical persistence:

```text
                  Application Business Logic & Services
                                    │
                                    ▼
                          Pogonian Data Runtime
       (Entity Schema Validation, Parameterized Queries, Prepared Cache)
                                    │
                                    ▼
                       Native Storage Implementation
             (Default: High-Performance SQLite WAL Mode)
```

- **SQLite WAL Mode**: Write-Ahead Logging (`PRAGMA journal_mode = WAL`), high-concurrency readers/writers, 5-second busy timeout.
- **Unified Query API**: `dr.Find(ctx, entity, filters...)`, `dr.FindByID(ctx, entity, id)`, `dr.Insert()`, `dr.Update()`, `dr.Delete()`.
- **Thread-Safe Pooling**: Configured connection pooling (25 max open, 5 max idle conns) with automatic directory creation.

---

## 📡 Hardware & Device Capability Model

Implemented in [`pkg/capability/capability.go`](file:///home/vraj/Desktop/Vruket/Pogonian/Pogonian/pkg/capability/capability.go):

Applications declare required or optional capabilities. The compiler's **Capability Validator** enforces compatibility at compile time:

```bash
pogn capabilities
```

- **Compile-Time Detection**: If an application requests `tv_remote` as `Required: true` but compiles for Web or Desktop, the compiler raises an immediate `ERROR` with actionable fallback advice.
- **Automated Permission Generation**: Generates native `<uses-permission>` tags for Android and privacy strings for iOS `Info.plist`.
- **Graceful Fallbacks**: Optional capabilities (`Required: false`) degrade gracefully using declared fallback actions.

---

## 🔄 Cross-Device State Continuity

Implemented in [`pkg/state/state.go`](file:///home/vraj/Desktop/Vruket/Pogonian/Pogonian/pkg/state/state.go):

Pogonian enables portable state handoff between heterogeneous platforms (e.g., Web → Android → Windows → Android TV):

```text
1. Origin Device (Web)       ──➔ Captures StateSnapshot (Route, Sync Fields)
2. State Engine              ──➔ Signs HMAC-SHA256 StateEnvelope with Expiration
3. Transport Network         ──➔ Transfers Portable Signed State Payload
4. Destination Device (TV)   ──➔ Verifies Cryptographic Signature & Integrity
5. Compatibility Engine      ──➔ Validates Semantic Version & Target Route
6. State Hydration           ──➔ Maps State into Local Model & Restores UI
```

- **Tamper-Evident Security**: Cryptographic HMAC verification requires a configured `APP_KEY` or `SESSION_SECRET`; insecure hard-coded fallback secrets are prohibited.
- **True SemVer Evaluation**: Numerical `Major.Minor.Patch` validation ensures state schema compatibility.
- **Local Scope Isolation**: Ephemeral fields are strictly filtered out of cross-device transfers.

---

## 🛡️ 5-Tier Backend Architecture (Go Foundation)

The backend maintains strict compile-time layer encapsulation verified by `pogn check`:

```text
┌────────────────────────────────────────────────────────┐
│ 1. Client Browser / Native Runtime                     │
└───────────────────────────┬────────────────────────────┘
                            │ HTTP / IPC Request
                            ▼
┌────────────────────────────────────────────────────────┐
│ 2. Radix Route Trie (pkg/router/trie.go - O(k))        │
└───────────────────────────┬────────────────────────────┘
                            │ Extracted Params (:id, [slug])
                            ▼
┌────────────────────────────────────────────────────────┐
│ 3. HTTP / Action Handler (server/<module>/handler.go)  │
│    - CSRF Token Validation                             │
│    - Form & Payload Decoding                           │
└───────────────────────────┬────────────────────────────┘
                            │ Typed DTO
                            ▼
┌────────────────────────────────────────────────────────┐
│ 4. Domain Service (server/<module>/service.go)         │
│    - Business Validation & Domain Rules                │
└───────────────────────────┬────────────────────────────┘
                            │ Domain Entity
                            ▼
┌────────────────────────────────────────────────────────┐
│ 5. Repository (server/<module>/repository.go)          │
│    - Parameterized Data Runtime Operations             │
└───────────────────────────┬────────────────────────────┘
                            │ Data Operations
                            ▼
┌────────────────────────────────────────────────────────┐
│ 6. Pogonian Data Runtime (SQLite WAL Mode Engine)      │
└────────────────────────────────────────────────────────┘
```

| Layer | Can Call | CANNOT Call |
| :--- | :--- | :--- |
| **Frontend UI (Web/Native)** | Client runtime, Native UI components | Direct SQL, Repositories, Database drivers |
| **`handler.go`** | Services, Response Writers | Raw SQL queries, Database drivers directly |
| **`service.go`** | Repositories, Domain Entities | HTTP Request/Response objects, Direct SQL |
| **`repository.go`**| Pogonian Data Runtime / DB Pool | HTTP objects, Services, Handlers |

---

## 🔒 Security Posture

1. **Argon2id Hashing**: Password storage adheres to modern RFC cryptographic standards (`$argon2id$v=19$m=65536,t=1,p=4$`).
2. **Fail-Closed CSRF**: Every mutable request (`POST`, `PUT`, `DELETE`, `PATCH`) must carry a valid cryptographic CSRF token or session match.
3. **Session Persistence**: Sessions are stored in the SQLite database to survive server reboots.
4. **Path Traversal Defense**: Static asset serving strictly checks folder boundaries preventing directory escapes.
5. **State Payload Cryptography**: Cross-device state transfers use HMAC-SHA256 envelope signatures.
