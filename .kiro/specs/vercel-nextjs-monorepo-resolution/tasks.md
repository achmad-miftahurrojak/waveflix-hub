# Implementation Plan

- [ ] 1. Write bug condition exploration test
  - **Property 1: Bug Condition** - Monorepo Package Resolution Failures
  - **CRITICAL**: This test MUST FAIL on unfixed code - failure confirms the bug exists
  - **DO NOT attempt to fix the test or the code when it fails**
  - **NOTE**: This test encodes the expected behavior - it will validate the fix when it passes after implementation
  - **GOAL**: Surface counterexamples that demonstrate the bug exists
  - **Scoped PBT Approach**: For deterministic bugs, scope the property to the concrete failing case(s) to ensure reproducibility
  - Test that `vercel build` fails with current monorepo configuration
  - Test that Next.js dependency tracing is limited to frontend directory only
  - Test that React version conflicts exist across workspace packages
  - Test that multiple lockfiles create non-deterministic dependency resolution
  - Run test on UNFIXED code
  - **EXPECTED OUTCOME**: Test FAILS (this is correct - it proves the bug exists)
  - Document counterexamples found to understand root cause
  - Mark task complete when test is written, run, and failure is documented
  - _Requirements: FR-1.1, FR-2.1, FR-3.1, FR-4.1_

- [ ] 2. Write preservation property tests (BEFORE implementing fix)
  - **Property 2: Preservation** - Development Workflow and Valid Configurations
  - **IMPORTANT**: Follow observation-first methodology
  - Observe behavior on UNFIXED code for non-buggy development workflows
  - Test that local development server (`npm run dev`) works correctly
  - Test that existing build scripts execute without errors in local environment
  - Test that workspace package isolation is maintained for internal dependencies
  - Write property-based tests capturing observed behavior patterns from Preservation Requirements
  - Property-based testing generates many test cases for stronger guarantees
  - Run tests on UNFIXED code
  - **EXPECTED OUTCOME**: Tests PASS (this confirms baseline behavior to preserve)
  - Mark task complete when tests are written, run, and passing on unfixed code
  - _Requirements: NFR-6.1, NFR-6.2_

- [ ] 3. Fix for Vercel Next.js monorepo package resolution

  - [ ] 3.1 Fix Next.js outputFileTracingRoot configuration
    - Update `waveflix-app/frontend/next.config.mjs` to point tracing root to monorepo root
    - Add path import to support relative path resolution: `import path from 'path'`
    - Change `outputFileTracingRoot: import.meta.dirname` to `outputFileTracingRoot: path.resolve(import.meta.dirname, '../../')`
    - _Bug_Condition: isBugCondition(config) where config.outputFileTracingRoot == "import.meta.dirname"_
    - _Expected_Behavior: expectedBehavior(result) where result.buildSucceeds == true AND result.dependencyTracingWorks == true_
    - _Preservation: Development workflow and valid Next.js configurations from design_
    - _Requirements: FR-1.1, FR-1.2_

  - [ ] 3.2 Synchronize React versions across workspaces
    - Update `waveflix-web/package.json` React version from 19.2.8 to match frontend
    - Update `waveflix-app/frontend/package.json` React version if needed for consistency
    - Ensure both `react` and `react-dom` versions match exactly
    - Run `npm install` to regenerate lockfile with consistent versions
    - _Bug_Condition: isBugCondition(config) where config.reactVersionsConflict == true_
    - _Expected_Behavior: expectedBehavior(result) where result.noRuntimeErrors == true AND result.singleReactVersion == true_
    - _Preservation: Existing React functionality and component behavior from design_
    - _Requirements: FR-2.1, FR-2.2_

  - [ ] 3.3 Remove duplicate lockfiles  
    - Delete `waveflix-app/frontend/package-lock.json`
    - Ensure only root-level `package-lock.json` exists for deterministic resolution
    - _Bug_Condition: isBugCondition(config) where config.multipleLockfiles == true_
    - _Expected_Behavior: expectedBehavior(result) where result.deterministicBuilds == true_
    - _Preservation: Dependency management and workspace isolation from design_
    - _Requirements: FR-3.1, FR-3.2_

  - [ ] 3.4 Clean up Vercel configuration
    - Review and update `vercel.json` to remove problematic service entries
    - Remove "backend" and "autoscaler" services that require unsupported Go runtime entrypoints
    - Ensure only frontend service is configured for deployment
    - _Bug_Condition: isBugCondition(config) where config.vercelServicesIncludeNonWeb == true_
    - _Expected_Behavior: expectedBehavior(result) where result.deploymentSucceeds == true_
    - _Preservation: Valid Vercel deployment configuration patterns from design_
    - _Requirements: FR-4.1, FR-4.2_

  - [ ] 3.5 Verify bug condition exploration test now passes
    - **Property 1: Expected Behavior** - Successful Monorepo Package Resolution
    - **IMPORTANT**: Re-run the SAME test from task 1 - do NOT write a new test
    - The test from task 1 encodes the expected behavior
    - When this test passes, it confirms the expected behavior is satisfied
    - Run bug condition exploration test from step 1
    - **EXPECTED OUTCOME**: Test PASSES (confirms bug is fixed)
    - _Requirements: Expected Behavior Properties from design_

  - [ ] 3.6 Verify preservation tests still pass
    - **Property 2: Preservation** - Development Workflow and Valid Configurations  
    - **IMPORTANT**: Re-run the SAME tests from task 2 - do NOT write new tests
    - Run preservation property tests from step 2
    - **EXPECTED OUTCOME**: Tests PASS (confirms no regressions)
    - Confirm all tests still pass after fix (no regressions)

- [ ] 4. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.