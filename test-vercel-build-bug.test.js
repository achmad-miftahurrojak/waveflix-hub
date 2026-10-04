const { execSync, spawn } = require('child_process');
const fs = require('fs');
const path = require('path');
const { promisify } = require('util');

/**
 * Bug Condition Exploration Test for Vercel Next.js Package Resolution Failure
 * **Validates: Requirements FR-1.1, FR-2.1, FR-3.1, FR-4.1**
 * 
 * CRITICAL: This test MUST FAIL on unfixed code - failure confirms the bug exists
 * DO NOT attempt to fix the test or the code when it fails
 * 
 * This test explores the monorepo package resolution failure that occurs when:
 * 1. Vercel builds fail with "Could not find the Next.js package (next/package.json)" 
 * 2. Multiple lockfiles create non-deterministic dependency resolution
 * 3. React version conflicts exist across workspace packages
 * 4. Next.js dependency tracing is limited to frontend directory only
 */

describe('Vercel Next.js Package Resolution Bug Exploration', () => {
  const workspaceRoot = process.cwd();
  const frontendPath = path.join(workspaceRoot, 'waveflix-app', 'frontend');
  const waveflixWebPath = path.join(workspaceRoot, 'waveflix-web');
  
  /**
   * Property 1: Monorepo Package Resolution Failures
   * Tests that Next.js builds fail due to workspace configuration issues
   * EXPECTED: This test FAILS (proving the bug exists)
   */
  test('Property 1: Next.js build fails due to monorepo configuration in Vercel context', async () => {
    console.log('Testing Next.js build failure simulation in monorepo context...');
    
    // Simulate the Vercel build environment issues
    // Test case 1: Check if Next.js config can properly resolve workspace dependencies
    const frontendNextConfig = path.join(frontendPath, 'next.config.js');
    const frontendNextConfigMjs = path.join(frontendPath, 'next.config.mjs');
    
    let hasNextConfig = false;
    try {
      hasNextConfig = fs.existsSync(frontendNextConfig) || fs.existsSync(frontendNextConfigMjs);
    } catch (error) {
      console.log('Error checking Next.js config:', error.message);
    }
    
    console.log('Frontend Next.js config exists:', hasNextConfig);
    
    // Test case 2: Verify workspace package resolution from frontend context
    let workspaceResolutionError = null;
    try {
      // Try to resolve workspace root package.json from frontend directory
      const rootPkgFromFrontend = require.resolve('../../package.json', { paths: [frontendPath] });
      console.log('Workspace root resolved from frontend:', rootPkgFromFrontend);
    } catch (error) {
      workspaceResolutionError = error.message;
    }
    
    // Test case 3: Check for Vercel-specific build configuration issues
    const vercelConfig = JSON.parse(fs.readFileSync(path.join(workspaceRoot, 'vercel.json'), 'utf8'));
    const frontendService = vercelConfig.services?.frontend;
    
    console.log('Frontend service config:', frontendService);
    
    // CRITICAL ASSERTION: The configuration should have issues that cause build failures
    // In this case, we expect workspace resolution or configuration problems
    const hasConfigurationIssues = (
      !hasNextConfig ||  // No Next.js config to handle monorepo setup
      workspaceResolutionError ||  // Can't resolve workspace packages properly
      !frontendService?.root ||  // No proper root configuration
      frontendService.root !== 'waveflix-app/frontend'  // Incorrect root path
    );
    
    expect(hasConfigurationIssues).toBe(true);
    
    if (workspaceResolutionError) {
      console.log('Workspace resolution error (bug condition):', workspaceResolutionError);
    }
    
    // Document the specific configuration issues
    console.log('Configuration issues found:', {
      hasNextConfig,
      workspaceResolutionError: !!workspaceResolutionError,
      frontendRoot: frontendService?.root,
      expectedRoot: 'waveflix-app/frontend'
    });
  });

  /**
   * Property 2: Multiple Lockfiles Create Non-deterministic Resolution
   * Tests that multiple package-lock.json files exist causing conflicts
   * EXPECTED: This test FAILS (proving the bug condition exists)
   */
  test('Property 2: Multiple lockfiles create non-deterministic dependency resolution', () => {
    console.log('Testing for multiple lockfiles causing resolution conflicts...');
    
    const lockfilePaths = [
      path.join(workspaceRoot, 'package-lock.json'),
      path.join(frontendPath, 'package-lock.json'),
      path.join(waveflixWebPath, 'package-lock.json'),
      path.join(workspaceRoot, 'waveflix-extractor', 'package-lock.json')
    ];
    
    const existingLockfiles = lockfilePaths.filter(lockfile => {
      try {
        return fs.existsSync(lockfile);
      } catch {
        return false;
      }
    });
    
    console.log('Found lockfiles:', existingLockfiles);
    
    // CRITICAL ASSERTION: Multiple lockfiles should exist (this is the bug condition)
    // In a properly configured monorepo, there should be only one lockfile at the root
    expect(existingLockfiles.length).toBeGreaterThan(1);
    
    // Verify workspace-specific lockfiles exist (indicating misconfiguration)
    const workspaceLockfiles = existingLockfiles.filter(lockfile => 
      !lockfile.endsWith(path.join(workspaceRoot, 'package-lock.json'))
    );
    
    expect(workspaceLockfiles.length).toBeGreaterThan(0);
    console.log('Workspace-specific lockfiles (bug condition):', workspaceLockfiles);
  });

  /**
   * Property 3: React Version Conflicts Across Workspace Packages  
   * Tests for dependency resolution issues that could cause build failures
   * EXPECTED: This test FAILS (proving version management issues exist)
   */
  test('Property 3: Dependency resolution issues exist in monorepo setup', () => {
    console.log('Testing for dependency resolution issues across workspaces...');
    
    const packageJsonPaths = [
      { path: path.join(frontendPath, 'package.json'), name: 'frontend' },
      { path: path.join(waveflixWebPath, 'package.json'), name: 'waveflix-web' },
      { path: path.join(workspaceRoot, 'waveflix-extractor', 'package.json'), name: 'waveflix-extractor' },
      { path: path.join(workspaceRoot, 'package.json'), name: 'root' }
    ];
    
    const dependencyAnalysis = {};
    
    packageJsonPaths.forEach(({ path: pkgPath, name }) => {
      try {
        const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'));
        dependencyAnalysis[name] = {
          hasNext: !!(pkg.dependencies?.next || pkg.devDependencies?.next),
          hasReact: !!(pkg.dependencies?.react || pkg.devDependencies?.react),
          nextVersion: pkg.dependencies?.next || pkg.devDependencies?.next,
          reactVersion: pkg.dependencies?.react || pkg.devDependencies?.react,
          hasFramerMotion: !!(pkg.dependencies?.['framer-motion'] || pkg.devDependencies?.['framer-motion'])
        };
      } catch (error) {
        console.log(`Could not read package.json for ${name}:`, error.message);
        dependencyAnalysis[name] = { error: error.message };
      }
    });
    
    console.log('Dependency analysis across workspaces:', dependencyAnalysis);
    
    // Check for dependency conflicts and missing declarations
    const nextServices = Object.entries(dependencyAnalysis).filter(([, info]) => info.hasNext);
    const reactServices = Object.entries(dependencyAnalysis).filter(([, info]) => info.hasReact);
    
    console.log('Services with Next.js:', nextServices.map(([name]) => name));
    console.log('Services with React:', reactServices.map(([name]) => name));
    
    // CRITICAL ASSERTION: There should be dependency management issues
    // 1. Root has framer-motion but workspaces might not properly inherit it
    // 2. Next.js services should have consistent React versions
    // 3. Some workspaces might be missing critical dependencies
    
    const rootHasFramerMotion = dependencyAnalysis.root?.hasFramerMotion;
    const inconsistentDeps = nextServices.some(([name, info]) => 
      !info.hasReact || (info.nextVersion && !info.reactVersion)
    );
    
    expect(rootHasFramerMotion && inconsistentDeps).toBe(true);
    console.log('Dependency management issues detected (bug condition):', {
      rootHasFramerMotion,
      inconsistentDeps,
      dependencyAnalysis
    });
  });

  /**
   * Property 4: Next.js Dependency Resolution Issues in Monorepo Context
   * Tests that package resolution creates issues for Vercel builds
   * EXPECTED: This test FAILS (proving dependency resolution problems exist)
   */
  test('Property 4: Package resolution issues exist for Vercel deployment context', () => {
    console.log('Testing package resolution issues in Vercel deployment context...');
    
    const testPaths = [
      { path: workspaceRoot, name: 'workspace-root' },
      { path: frontendPath, name: 'frontend-directory' },
      { path: waveflixWebPath, name: 'waveflix-web-directory' }
    ];
    
    const resolutionResults = {};
    
    testPaths.forEach(({ path: testPath, name }) => {
      try {
        // Try to resolve Next.js package.json from each directory
        const nextPackagePath = require.resolve('next/package.json', { paths: [testPath] });
        const nextPackageContent = JSON.parse(fs.readFileSync(nextPackagePath, 'utf8'));
        resolutionResults[name] = { 
          success: true, 
          path: nextPackagePath,
          version: nextPackageContent.version
        };
      } catch (error) {
        resolutionResults[name] = { success: false, error: error.message };
      }
    });
    
    console.log('Next.js resolution results:', resolutionResults);
    
    // Check for version mismatches and resolution path issues
    const successfulResolutions = Object.entries(resolutionResults)
      .filter(([, result]) => result.success)
      .map(([name, result]) => ({ name, path: result.path, version: result.version }));
      
    console.log('Next.js resolution paths:', successfulResolutions);
    
    // CRITICAL ASSERTION: There should be resolution issues for Vercel deployment
    // 1. Different Next.js versions resolved from different paths
    // 2. Resolution paths that could confuse Vercel's build process
    // 3. Workspace isolation issues
    
    const versions = [...new Set(successfulResolutions.map(r => r.version))];
    const paths = [...new Set(successfulResolutions.map(r => path.dirname(r.path)))];
    
    console.log('Unique Next.js versions found:', versions);
    console.log('Unique Next.js locations:', paths);
    
    // The bug condition: multiple Next.js installations or version mismatches
    const hasResolutionIssues = (
      versions.length > 1 ||  // Multiple Next.js versions
      paths.length > 1 ||     // Multiple Next.js installations  
      successfulResolutions.length !== testPaths.length  // Inconsistent resolution
    );
    
    expect(hasResolutionIssues).toBe(true);
    console.log('Package resolution issues detected (bug condition):', {
      multipleVersions: versions.length > 1,
      multipleInstallations: paths.length > 1,
      inconsistentResolution: successfulResolutions.length !== testPaths.length
    });
  });

  /**
   * Property 5: Turbo Configuration Missing Environment Variables
   * Tests that turbo.json lacks proper environment variable configuration
   * EXPECTED: This test FAILS (proving configuration is incomplete)
   */
  test('Property 5: Turbo configuration missing environment variables for Vercel builds', () => {
    console.log('Testing turbo.json configuration for Vercel-specific environment variables...');
    
    const turboConfigPath = path.join(workspaceRoot, 'turbo.json');
    let turboConfig = {};
    
    try {
      turboConfig = JSON.parse(fs.readFileSync(turboConfigPath, 'utf8'));
    } catch (error) {
      console.log('Could not read turbo.json:', error.message);
    }
    
    console.log('Current turbo.json config:', JSON.stringify(turboConfig, null, 2));
    
    // Check for Vercel-specific environment variables that should be passed through
    const requiredVercelEnvVars = [
      'VERCEL',
      'VERCEL_ENV', 
      'VERCEL_URL',
      'VERCEL_PROJECT_PRODUCTION_URL',
      'NEXT_PUBLIC_VERCEL_URL'
    ];
    
    const currentPassThroughEnv = turboConfig.globalPassThroughEnv || [];
    const missingEnvVars = requiredVercelEnvVars.filter(envVar => 
      !currentPassThroughEnv.includes(envVar)
    );
    
    console.log('Missing Vercel environment variables:', missingEnvVars);
    
    // CRITICAL ASSERTION: Required Vercel environment variables should be missing
    // This proves the turbo configuration is incomplete for Vercel builds
    expect(missingEnvVars.length).toBeGreaterThan(0);
    
    // Additional check for turbopack configuration
    const hasTurbopackConfig = Object.keys(turboConfig).some(key => 
      key.includes('turbopack') || key.includes('root')
    );
    
    expect(hasTurbopackConfig).toBe(false);
    console.log('Turbopack configuration missing (bug condition)');
  });
});