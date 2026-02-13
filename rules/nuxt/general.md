## Stack

- Nuxt: 4.x

---

# AI Instructions for @spilnu/core and Related Projects

This document provides coding guidelines and preferences for AI assistants working on projects that use `@spilnu/core` or similar codebases.

## Project Context

- **Framework**: Nuxt 4 with Vue 3 and TypeScript
- **Package Type**: Nuxt module providing core functionality for multiple brands
- **Architecture**: Monorepo-style module with feature sub-modules (`modules/`)
- **Target Platforms**: Desktop and mobile web applications (responsive design)
- **Primary Markets**: Danish and English gaming/casino applications
- **Styling**: SCSS with mobile-first approach
- **Testing**: Vitest 4.x with `@nuxt/test-utils` for unit and Nuxt integration tests
- **Package Manager**: Bun (private registry scopes for `@cego`, `@spilnu`, `@engage`)

## AI Assistant Tools & Resources

### Nuxt MCP Server

**Always use the Nuxt MCP Server when available** for accurate, up-to-date Nuxt documentation:

- **MCP Server**: `@nuxt`
- **Endpoint**: `https://nuxt.com/mcp`
- **Setup**: Configure in VS Code via `.vscode/settings.json` or user MCP settings
- **Capabilities**:
    - Search official Nuxt documentation (Nuxt Core, Nuxt UI, Nuxt Content, NuxtHub)
    - List and discover Nuxt modules with stats and compatibility info
    - Get accurate API references and best practices

**When to use:**

- Questions about Nuxt features, APIs, or configuration
- Looking up best practices for Nuxt-specific functionality
- Checking Nuxt module compatibility and installation
- Understanding Nuxt UI components usage
- NuxtHub database, blob, or KV store implementation

**Example queries:**

```
@nuxt How do I configure a custom Nuxt module?
@nuxt What are the best practices for SEO meta tags in Nuxt?
@nuxt Show me Nuxt UI Button component props
@nuxt How to use useAsyncData with error handling?
```

**Important**: Always prefer MCP-sourced documentation over assumptions or outdated information. If the MCP server is not available, clearly indicate you're working without live documentation access.

## Code Style & Formatting

### Linting & Code Quality

- Use `@cego/eslint-config-nuxt` for ESLint configuration (flat config format via `eslint.config.mjs`)
- Run `bun run lint:fix` before committing
- Follow TypeScript strict mode conventions
- Enable `noUncheckedIndexedAccess` in TypeScript config

### Indentation & Spacing

**Standard:**

- 4 spaces for TypeScript/JavaScript/Vue/SCSS/CSS files
- No trailing whitespace
- End files with a single newline

### Line Length

- No strict limit, but prioritize readability
- Break long lines sensibly at logical points
- Keep function signatures readable

## Vue Component Structure

### File Naming

- Use PascalCase for component files: `AccountBalanceOverview.vue`, `UIButton.vue`,
- Prefix UI components with `UI`: `UIDialog.vue`, `UIInput.vue`
- Use kebab-case for composable files: `usePlayerAccountClient.ts`, `useApi.ts`

### Component Order

**Standard structure:**

```vue
<script setup lang="ts">
// 1. Imports (grouped by source)
// 2. Type/Interface definitions (if small and component-specific)
// 3. Props definition with `defineProps` or `withDefaults`
// 4. Emits definition with `defineEmits`
// 5. Composables
// 6. Reactive state
// 7. Computed properties
// 8. Functions
// 9. Lifecycle hooks
// 10. Provide/Inject
// 11. defineExpose
</script>

<template>
    <!-- Component template -->
</template>

<style lang="scss" scoped>
// Component styles
</style>
```

### Script Setup Syntax

**Always use:**

- `<script setup lang="ts">` (NOT Options API)
- TypeScript for all script blocks
- Explicit type annotations for props

**Example:**

```vue
<script setup lang="ts">
export interface UIButtonProps {
    variant?: 'primary' | 'secondary' | 'danger'
    disabled?: boolean
    loading?: boolean
}

const props = withDefaults(defineProps<UIButtonProps>(), {
    variant: 'primary',
    disabled: false,
    loading: false
})
</script>
```

### Props Definition

- Use TypeScript interface + `defineProps<T>()`
- Use `withDefaults(defineProps<UIButtonProps>() { ... })` when setting defaults is needed

**Example**

```typescript
export interface UILoaderProps {
    bg?: boolean
    color?: string
    inline?: boolean
    width?: string
    height?: string
    text?: string | false
}

const props = defineProps<UILoaderProps>()
```

## TypeScript Conventions

### Type Definitions

- Store shared types in `src/runtime/types/`
- Use type imports: `import type { ... } from '...'`
- Prefer interfaces over types for object shapes
- Define component-specific interfaces and extract to separate type file in `types/components/` if used by multiple components
- Generated OpenAPI client types live in `src/runtime/types/clients/` (auto-generated via `bun run generate:clients`)

### Type Naming

- Prefix component prop interfaces with component name: `UIButtonProps`, `UIDialogProps`
- Use descriptive names: `UseQueryReturn`, `ApiResponse`, `ServiceOptions`
- Suffix return types with `Return`: `UseDisplayReturn`, `UseApiReturn`
- Suffix options with `Options`: `UseQueryOptions`, `UseFormatOptions`

### Imports

**Import order:**

1. Vue core imports
2. Nuxt imports (`#imports`, `#app`)
3. Third-party packages
4. Local type imports (`@core/types`)
5. Relative imports

**Use auto-imports:**

- Nuxt auto-imports components, composables and Vue.js APIs to use across the application
- Nuxt utilities are auto-imported
- Vue utilities are auto-imported (ref, computed, watch, etc.)

**Example:**

```typescript
import type { TransitionProps, HTMLAttributes } from 'vue'
import type { UIDialogInstance } from '@core/types'
import { onUnmounted, onMounted, watch, computed, ref } from '#imports'
```

### Utility Libraries

**Always prefer existing utilities from established libraries over writing custom implementations:**

**@vueuse/core** - For Vue-specific utilities and composables:

- Use `@vueuse/core` for common reactive utilities: `useToggle`, `useDebounceFn`, `useThrottleFn`, `useEventListener`, etc.
- Prefer VueUse composables for DOM interactions, localStorage, media queries, etc.
- Examples: `useStorage`, `useMediaQuery`, `useIntersectionObserver`, `useElementSize`

**es-toolkit** - For general JavaScript utilities:

- Use `es-toolkit` for array/object manipulation: `pick`, `omit`, `groupBy`, `chunk`, `debounce`, `throttle`
- Prefer es-toolkit over lodash or custom utilities for common operations
- Examples from codebase: `pick(props, allowedKeys)`, `groupBy(items, 'category')`

**zod** - For schema validation:

- Use `zod` for runtime data validation and schema definitions
- Prefer zod over manual validation logic

**vee-validate** - For form validation:

- Use `vee-validate` for form validation in Vue components

**maska** - For input masking:

- Use `maska` for input masking (e.g., phone numbers, dates)

**Why:**

- Battle-tested, optimized implementations
- Smaller bundle sizes (especially es-toolkit)
- Type-safe TypeScript definitions
- Maintained by the community
- Reduces custom code maintenance burden

**When to write custom utilities:**

- Business logic specific to Spilnu/gaming domain
- Complex brand-specific transformations
- When no suitable library function exists

**Example - Prefer library utilities:**

```typescript
// Avoid writing custom implementations
const picked = Object.keys(props)
    .filter(key => allowedKeys.includes(key))
    .reduce((obj, key) => ({ ...obj, [key]: props[key] }), {})

// Use es-toolkit
import { pick } from 'es-toolkit'
const picked = pick(props, allowedKeys)

// Avoid custom debounce
let timeout: NodeJS.Timeout
const handleSearch = (value: string) => {
    clearTimeout(timeout)
    timeout = setTimeout(() => search(value), 300)
}

// Use @vueuse/core
import { useDebounceFn } from '@vueuse/core'
const handleSearch = useDebounceFn((value: string) => search(value), 300)
```

## SCSS/CSS Conventions

### Mobile-First Approach

**Always design mobile-first**, then add larger breakpoint styles:

```scss
.container {
    // Mobile styles (default)
    padding: 15px;

    // Tablet
    @media (min-width: $screen-md) {
        padding: 30px;
    }

    // Desktop
    @media (min-width: $screen-lg) {
        padding: 40px;
    }
}
```

### Media Query Mixins

**Use provided mixins instead of raw media queries:**

```scss
.container {
    padding: 15px;

    @include tablet() {
        padding: 30px;
    }

    @include desktop() {
        padding: 40px;
    }

    @include desktop-xl() {
        max-width: 1024px;
    }
}
```

### Variables & Functions

- Use SCSS variables from `variables.scss` (auto-imported)
- Use the `z()` function for z-index values: `z-index: z(dialog)`
- Avoid using deprecated Sass functions (e.g., `darken`, `lighten`, `adjust-color`). Use modern alternatives from the `sass:color` module like `color.adjust()`, `color.mix()`.
- Use math functions: `math.floor()`, `math.ceil()`

**Required imports (already available globally):**

```scss
@use "sass:math";
@use "sass:color";
```

### BEM-Like Naming

- We prefer BEM-like naming with `&` for nested elements, but allow simple class names when component is small and straightforward

**Example:**

```scss
.dialog {
    // Block

    &__header {
        // Element
    }

    &__footer {
        // Element
    }

    &--wide {
        // Modifier
    }

    &--bounce-down {
        // Modifier
    }
}
```

### Scoped Styles

- Always use `<style lang="scss" scoped>` in components
- Use `:deep()` for styling elements rendered by components or slots within the component
- Avoid `::v-deep` (deprecated)

**Example:**

```vue
<template>
    <div v-show="modelValue" class="'alert'">
        <UIButton
            v-if="closeable"
            as="close"
            class="alert__btn--close"
            @click.prevent="onCloseClick"
        >
            <!-- ... -->
        </UIButton>

        <div class="alert__message">
            <!-- ... -->
            <div>
                <slot>
                    <p>{{ message }}</p>
                </slot>
            </div>
        </div>
    </div>
</template>
<style lang="scss" scoped>
.alert {
    padding: 12px;
    position: relative;
    font-size: $font-size-sm;
    z-index: 3;

    // Target child p elements in slot
    :slotted(p) {
        color: inherit;
        margin: 0 0 5px;
        padding: 0;
        font-size: inherit;
    }

    // Target child element in UIButton
    :deep(.btn__content) {
        font-size: inherit;
    }
}
</style>
```

## Composables

### File Naming

- Prefix with `use`: `usePlayerAccountClient.ts`, `useApi.ts`, `useDayjs.ts`
- Place in `src/runtime/composables/`
- Data-fetching client composables go in `src/runtime/composables/data/`
- Data-layer primitives (useQuery, useMutation, etc.) live in `src/runtime/composables/data-layer/`

### Composable Structure

**Standard pattern:**

```typescript
import { createSharedComposable } from '#imports'

export const useExample = createSharedComposable(() => {
    // State
    const state = ref(null)

    // Computed
    const computed = computed(() => state.value)

    // Functions
    function doSomething() {
        // ...
    }

    // Return
    return {
        state,
        computed,
        doSomething
    }
})
```

- Use `createSharedComposable` for singleton composables

### Type Safety

- Always provide return type for exported composables
- Use explicit types for reactive state
- Type all function parameters

**Example:**

```typescript
export interface UseDialogReturn {
    isOpen: Ref<boolean>
    open: () => void
    close: () => void
}

export const useDialog = (): UseDialogReturn => {
    const isOpen = ref<boolean>(false)

    function open() {
        isOpen.value = true
    }

    function close() {
        isOpen.value = false
    }

    return {
        isOpen,
        open,
        close
    }
}
```

## API Integration

### Data Layer Architecture

The project has a custom data-fetching layer in `src/runtime/composables/data-layer/` built on top of Nuxt's `useAsyncData`. It provides:

| Composable | Purpose |
|---|---|
| `useQuery` | Primary query composable wrapping `useAsyncData` with cache TTL, `enabled` ref/getter, abort controller |
| `useLazyQuery` | Convenience wrapper calling `useQuery` with `lazy: true, server: false` |
| `useInfiniteQuery` | Pagination support with `fetchNextPage()`/`fetchPreviousPage()` |
| `useMutation` | Standalone mutation state management with `mutate()`/`mutateAsync()` and lifecycle hooks |
| `useQueryClient` | Query management: `invalidateQueries()`, `refetchQueries()`, `removeQueries()`, `matchQueries()` |
| `useQueryCache` | TTL-based cache for `getCachedData` integration |

### useApi vs typed clients with useOpenapi

Some endpoints / services are not typed, which means we should use `useApi`. Otherwise in most cases we should use a typed client using `useOpenapi` and generate from openapi spec in `configs/clients/<some-client>` and generate the types using `bun run generate:clients`.

**`useApi`** - Low-level `$fetch`-based client:
- Wraps `ofetch`'s `$fetch.create()` with preconfigured interceptors
- Handles cookie forwarding (SSR), auth redirect on 401, service base URL resolution
- Supports `enableLegacySupport` for monolith endpoints

**`useOpenapi`** - Type-safe OpenAPI client:
- Wraps `openapi-fetch`'s `createClient<Paths>()` with typed paths
- Same middleware pattern: cookie forwarding, 401 redirect, service URL resolution
- Throws `OpenapiError` on non-OK responses

### Client Composables

`useQuery` and `useMutation` should be colocated based on services in client composables e.g. `usePlayerAccountClient` for the `player-account` service. All client composables live in `src/runtime/composables/data/`.

```typescript
// In usePlayerAccountClient.ts

export const usePlayerAccountClient = () => {
   const { site } = useCoreConfig()
   const client = useOpenapi<PlayerAccountClient.paths>({
       baseUrl: '/gateway/player-account/api',
       service: 'playerAccount'
   })
   const session_expiry = useCookie('session_expiry', {
       readonly: true
   })

   function useGetUserInfo(options: UseQueryOverrideOptions = {}) {
     const query = useQuery(async () => {
       const { data } = await client.GET("/public/user/info")
       return data
     }, {
         server: false, // For data that should not be available on initial render
         lazy: true, // For data that should not be available on initial render
         ...options,
         key: "playerAccount:userInfo",
         enabled: () => options.enabled !== undefined ? toValue(options.enabled) && !!session_expiry.value : !!session_expiry.value
     })
   }

   function useUpdateOccupation() {
     return useMutation(async (body: NonNullable<PlayerAccountClient.paths['/public/user/occupation']['post']['requestBody']>['content']['application/json']) => {
         const { data } = await client.POST("/public/user/occupation", {
             body
         })

         return data
     })
   }
}
```

Don't handle errors inside useMutation, since useMutation wraps the handler function in a try/catch and manages error-handling, request status and more.

### Using useLazyQuery

**For data that loads after initial render:**

```typescript
const { data, error, pending } = useLazyQuery(async () => {
    const { data } = await client.GET('/some/endpoint')
    return data
}, {
    key: 'myQuery:lazyData'
})
```

### OpenAPI Client Generation

OpenAPI specs are stored in `configs/clients/` as JSON files. The `redocly.yml` file maps specs to output files in `src/runtime/types/clients/`. Generate with:

```bash
bun run generate:clients
```

## Nuxt Module Development

### Module Structure

- Main module file: `src/module.ts`
- Runtime code: `src/runtime/`
- Type definitions: `src/runtime/types/`
- Build config: `build.config.ts`
- Feature sub-modules: `modules/<feature>/module.ts`

### Feature Sub-Modules

The project uses feature sub-modules in `modules/`, each following the same runtime structure as `src/runtime/`:

| Module | Purpose |
|---|---|
| `modules/game/` | Game tiles, lists, search, jackpots, bonus play |
| `modules/nimbus/` | Hero sections, bingo, blog, press, Strapi CMS, account flows |
| `modules/payment/` | Deposits, withdrawals, money limits |
| `modules/promotion/` | Signup bundles/offers |
| `modules/responsible-gambling/` | Responsible gambling features |

Each sub-module has its own `module.ts` entry, `runtime/` directory with components, composables, types, i18n locales, and other resources. They are registered in the root `nuxt.config.ts`.

### Module Options

- Define options interface in `src/runtime/types/global.ts`
- Provide sensible defaults in `src/options/index.ts`
- Make options available via `useCoreConfig()`

### Adding Components

```typescript
addComponentsDir({
    path: resolve(runtimeDir, 'components'),
    pathPrefix: true // Enables UI/Button.vue -> UIButton
})
```

### Adding Composables

```typescript
addImportsDir([
    resolve(runtimeDir, 'composables'),
    resolve(runtimeDir, 'composables', 'data'),
])
```

### Adding Plugins

```typescript
addPlugin({
    src: resolve(runtimeDir, 'plugins/dev'),
    mode: 'client' // or 'server' or omit for both
})
```

## Testing

### Test Configuration

Tests use a shared `defineVitestConfig()` from `test-utils/config.ts` which creates a multi-project Vitest setup:

| Project | Glob Pattern | Environment | Purpose |
|---|---|---|---|
| **unit** | `test/{e2e,unit}/**/*.{test,spec}.ts` | `node` | Pure logic tests (no Nuxt/Vue context) |
| **nuxt** | `test/nuxt/**/*.{test,spec}.ts` | `nuxt` (via `@nuxt/test-utils`) | Tests requiring Nuxt runtime (`#imports`, `#app`) |

The test-utils config is also exported as `@spilnu/core/test-utils/config` for reuse in consuming packages.

### Test File Naming

- Unit tests: `*.test.ts` or `*.spec.ts`
- Place in `test/unit/` for pure logic or `test/nuxt/` for Nuxt-dependent tests
- Mirror the source file structure (e.g., `test/unit/composables/`, `test/nuxt/utils/`)

### Test Structure

**Unit tests (node environment):**

```typescript
import { describe, it, expect } from 'vitest'

describe('ComponentName', () => {
    it('should do something', () => {
        // Arrange
        const input = 'test'

        // Act
        const result = functionToTest(input)

        // Assert
        expect(result).toBe('expected')
    })
})
```

**Nuxt tests (with auto-imports and Nuxt context):**

```typescript
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mockNuxtImport } from '@nuxt/test-utils/runtime'

// Mock auto-imported composables
mockNuxtImport('useSomeComposable', () => vi.fn(() => ({ ... })))

describe('useMyComposable', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })

    it('should handle data', () => {
        // Import from source path
        const { useMyComposable } = await import('../../../src/runtime/composables/useMyComposable')
        // ...
    })
})
```

### Test Utilities

The `test-utils/` directory exports shared mock infrastructure:

- `@spilnu/core/test-utils` - Mock helpers (e.g., `mocks.mockAll()` for plugin mocks)
- `@spilnu/core/test-utils/config` - Shared `defineVitestConfig()` function

### Running Tests

```bash
bun run test          # Run all tests
bun run test:dev      # Watch mode
bun run test:coverage # With Istanbul coverage
```

## Commit & Documentation

### Commit Messages

- Use conventional commits format with clear, descriptive messages in imperative mood

**Examples:**

```
feat: add new UIDialog component
fix: resolve z-index issue in drawer
refactor: extract theme logic to directive
docs: update API documentation
test: add tests for usePlayerAccountClient composable
```

### Code Comments

- Add JSDoc comments for exported functions and composables
- Explain "why" not "what" in inline comments
- Use TODO comments for future improvements: `// TODO: Add pagination support`
- Avoid adding types in JSDoc, since typescript should manage the types

### TypeScript Documentation

```typescript
/**
 * Formats a number as currency
 *
 * @param value - The numeric value to format
 * @param options - Formatting options
 * @returns Formatted currency string
 *
 * @example
 * asMoney(1234.56) // "kr. 1.234,56"
 */
export const asMoney = (value: number, options?: FormatOptions): string => {
    // ...
}
```

## Common Patterns

### Provide/Inject

```typescript
// Provider component
const dialog = {
    isOpen,
    open,
    close
}
provide(UIDIALOG_KEY, dialog)

// Consumer composable
export const useDialog = () => {
    const dialog = inject(UIDIALOG_KEY, undefined)

    if (!dialog) {
        throw new Error('No dialog available in context!')
    }

    return dialog
}
```

### Computed Properties

```typescript
// Use computed for derived state
const fullName = computed(() => `${firstName.value} ${lastName.value}`)

// Use computed for reactive references
const route = useRoute()
const dialogInQuery = computed(() => !!route.query.dialog)
```

### Watchers

- Use `watch` for side effects based on reactive changes

```typescript
// Watch specific sources
watch([source1, source2], ([newVal1, newVal2]) => {
    // React to changes
})

// Watch with options
watch(source, (newVal, oldVal) => {
    // ...
}, {
    immediate: true,
    deep: true
})
```

## Security & Performance

### Security Headers

- All CSP rules are configured in `src/module.ts`
- Use `getTrustedOrigins()` helper for environment-based domains
- Never expose secrets in public module options (they become public)

### Performance

- Use `lazy: true` for non-critical data (below the fold)
- Use `server: false` when data is client-only and for non-critical data (below the fold)
- Leverage query caching with TTL
- Use `v-once` for static content
- Use `v-memo` for expensive renders

## Brand-Specific Considerations

### Multi-Brand Support

- Define brand-specific values in SCSS variables
- Use runtime config for brand-specific API endpoints
- Support multiple jurisdictions (DK and UK)
- Handle brand-specific translations

### Theming

- Theme system lives in `src/runtime/theme/` with theme definitions in `themes/`
- Make colors configurable via SCSS variables
- Use CSS custom properties for runtime theming
- Theme is configurable via `moduleOptions.theme`

### Internationalization

- i18n is managed via `@nuxtjs/i18n`
- Core locale files live in `src/runtime/i18n/locales/`
- Each sub-module has its own `i18n/locales/` directory
- Supports Danish (da) and English (en)

## Questions & Edge Cases

When encountering ambiguous situations:

1. **Check existing code patterns** in similar components
2. **Refer to this document** for guidance
3. **Ask for clarification** with specific options:
    - "Should I use OPTION A or OPTION B for this case?"
    - Provide context and your reasoning
4. **Prioritize consistency** with existing codebase over personal preference

## Version Information

- **Nuxt**: 4.x
- **Vue**: 3.x
- **TypeScript**: 5.x
- **Node**: >= 24.x
- **Package Manager**: Bun
