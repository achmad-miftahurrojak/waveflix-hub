# Design: Vercel Next.js Package Resolution Fix for Monorepo

## Bug Analysis

### Bug Condition
The monorepo package resolution bug occurs when **any** of the following conditions are true:

```pseudocode
isBugCondition(config) = 
  (config.outputFileTracingRoot == "import.meta.dirname") OR
  (config.reactVersionsConflict == true) OR  
  (config.multipleLockfiles == true) OR
  (config.vercelServicesIncludeNonWeb == true)
```

**Concrete Examples:**
- `outputFileTracingRoot: import.meta.dirname` in `next.config.mjs` (limits tracing to frontend directory)
- React 19.0.0 in frontend vs React 19.2.8 in waveflix-web
- Both root `package-lock.json` and `waveflix-app/frontend/package-lock.json` exist
- `vercel.json` includes "backend" service with Go runtime requiring entrypoint

### Current Defective Behavior
When the bug condition is satisfied:
1. **Next.js build fails** to trace dependencies from root `node_modules`
2. **Vercel deployment errors** with "must specify entrypoint" for Go services
3. **Runtime errors** occur due to duplicate React instances
4. **Non-deterministic builds** happen due to conflicting lockfiles

### Expected Behavior Properties

```pseudocode
expectedBehavior(result) =
  (result.buildSucceeds == true) AND
  (result.deploymentSucceeds == true) AND  
  (result.noRuntimeErrors == true) AND
  (result.deterministicBuilds == true)
```

**Specific Expected Outcomes:**
- `vercel build` completes without package resolution errors
- Next.js traces dependencies correctly across workspace boundaries  
- Single React version used throughout the application
- Consistent dependency resolution in all environments

## Preservation Requirements

The fix must preserve existing behavior for **all non-buggy configurations**:

### Development Workflow Preservation
- Local development server (`npm run dev`) continues to work
- Hot module replacement and fast refresh remain functional
- Existing build scripts and commands work unchanged

### Dependency Management Preservation  
- Workspace package isolation is maintained where appropriate
- Internal package dependencies continue to resolve correctly
- External package versions remain stable (except for React sync)

### Configuration Preservation
- All valid Next.js configuration options remain functional
- Turbo.json pipeline configuration stays unchanged
- Package.json workspace definitions are preserved

```pseudocode
preservationCondition(config) = 
  (config.localDevWorks == true) AND
  (config.workspaceIsolation == maintained) AND
  (config.existingScriptsWork == true)
```

## Technical Solution

### 1. Fix outputFileTracingRoot Configuration

**File:** `waveflix-app/frontend/next.config.mjs`

**Current (Buggy):**
```javascript
outputFileTracingRoot: import.meta.dirname,
```

**Fixed:**
```javascript  
outputFileTracingRoot: path.resolve(import.meta.dirname, '../../'),
```

**Rationale:** Points Next.js file tracing to monorepo root, enabling dependency resolution from root `node_modules`.

### 2. Synchronize React Versions

**Files to Update:**
- `waveflix-app/frontend/package.json` 
- `waveflix-web/package.json`

**Action:** Update all React dependencies to use the same version (19.2.8 - the latest).

**Verification:** Run `npm ls react` to confirm no version conflicts.

### 3. Remove Duplicate Lockfiles

**Action:** Delete `waveflix-app/frontend/package-lock.json`

**Rationale:** Ensures single source of truth for dependency resolution from root lockfile.

### 4. Clean Up Vercel Configuration

**File:** `vercel.json`

**Action:** Remove non-web services (backend, autoscaler) from services configuration or provide proper entrypoints.

**Rationale:** Prevents Vercel deployment errors for unsupported service configurations.

## Implementation Approach

### Phase 1: Configuration Fixes
1. Update `next.config.mjs` with correct `outputFileTracingRoot`
2. Add proper path import to support relative path resolution

### Phase 2: Dependency Synchronization  
1. Update React versions in all workspace package.json files
2. Run `npm install` to regenerate lockfile with consistent versions
3. Remove workspace-specific lockfiles

### Phase 3: Vercel Configuration Cleanup
1. Remove problematic service entries from `vercel.json`
2. Ensure only frontend service is configured for deployment

### Phase 4: Validation
1. Test local build process  
2. Test Vercel deployment
3. Verify no runtime errors in deployed application

## Risk Assessment

**Low Risk Changes:**
- Updating React versions (backwards compatible within major version)
- Removing duplicate lockfiles (improves consistency)

**Medium Risk Changes:**  
- Modifying `outputFileTracingRoot` (could affect bundle size)
- Vercel configuration changes (affects deployment process)

**Mitigation:**
- Test local builds before deployment
- Keep backup of original configurations
- Deploy to preview environment first