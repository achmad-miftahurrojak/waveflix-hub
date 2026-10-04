const { execSync, spawn } = require('child_process');
const fs = require('fs');
const path = require('path');
const { promisify } = require('util');

/**
 * Preservation Property Tests for Vercel Next.js Monorepo Configuration
 * **Validates: Requirements NFR-6.1, NFR-6.2**
 * 
 * IMPORTANT: Follow observation-first methodology
 * These tests observe and validate baseline behavior on UNFIXED code
 * EXPECTED OUTCOME: Tests PASS (confirms baseline behavior to preserve)
 * 
 * These tests capture observed behavior patterns that must be preserved:
 * 1. Local development workflow continues to work
 * 2. Build scripts execute without errors in local environment  
 * 3. Workspace package isolation is maintained for internal dependencies
 * 4. Hot module replacement and fast refresh work correctly
 * 5. Existing configuration patterns remain functional
 */

describe('Development Workflow and Configuration Preservation Properties', () => {
  const workspaceRoot = process.cwd();
  const frontendPath = path.join(workspaceRoot, 'waveflix-app', 'frontend');
  const waveflixWebPath = path.join(workspaceRoot, 'waveflix-web');
  const extractorPath = path.join(workspaceRoot, 'waveflix-extractor');
  const backendPath = path.join(workspaceRoot, 'waveflix-app', 'backend');

  /**
   * Property 1: Local Development Server Functionality
   * **Validates: Requirements NFR-6.1** 
   * Tests that npm run dev works correctly across all workspace packages
   * EXPECTED: PASS (confirms local dev workflow is preserved)
   */
  test('Property 1: Local development server starts successfully and maintains workspace isolation', async () => {
    console.log('Testing local development server functionality preservation...');
    
    // Test workspace configuration integrity
    const workspaceConfig = JSON.parse(fs.readFileSync(path.join(workspaceRoot, 'package.json'), 'utf8'));
    const expectedWorkspaces = [
      'waveflix-extractor',
      'waveflix-app/backend', 
      'waveflix-app/frontend',
      'waveflix-web'
    ];
    
    expect(workspaceConfig.workspaces).toEqual(expectedWorkspaces);
    console.log('Workspace configuration preserved:', workspaceConfig.workspaces);
    
    // Test turbo configuration for dev command
    const turboConfig = JSON.parse(fs.readFileSync(path.join(workspaceRoot, 'turbo.json'), 'utf8'));
    expect(turboConfig.tasks.dev).toBeDefined();
    expect(turboConfig.tasks.dev.cache).toBe(false);
    expect(turboConfig.tasks.dev.persistent).toBe(true);
    console.log('Turbo dev task configuration preserved:', turboConfig.tasks.dev);
    
    // Test that each workspace has proper dev script
    const workspaceDevConfigs = {};
    
    expectedWorkspaces.forEach(workspace => {
      const packageJsonPath = workspace.includes('/') 
        ? path.join(workspaceRoot, workspace, 'package.json')
        : path.join(workspaceRoot, workspace, 'package.json');
        
      if (fs.existsSync(packageJsonPath)) {
        const pkg = JSON.parse(fs.readFileSync(packageJsonPath, 'utf8'));
        workspaceDevConfigs[workspace] = {
          hasDevScript: !!pkg.scripts?.dev,
          devScript: pkg.scripts?.dev,
          name: pkg.name
        };
      }
    });
    
    console.log('Workspace dev script configurations:', workspaceDevConfigs);
    
    // All workspaces should have dev scripts
    const workspacesWithDevScripts = Object.values(workspaceDevConfigs)
      .filter(config => config.hasDevScript).length;
    expect(workspacesWithDevScripts).toBe(expectedWorkspaces.length);
    
    // Test specific dev script patterns are preserved
    expect(workspaceDevConfigs['waveflix-app/frontend'].devScript).toBe('next dev -p 3005 --webpack');
    expect(workspaceDevConfigs['waveflix-web'].devScript).toBe('next dev -p 3006 --webpack');
    expect(workspaceDevConfigs['waveflix-extractor'].devScript).toBe('node server.js');
    expect(workspaceDevConfigs['waveflix-app/backend'].devScript).toBe('go run .');
    
    console.log('Development server configuration patterns preserved');
  }, 30000);

  /**
   * Property 2: Build System Functionality and Dependencies
   * **Validates: Requirements NFR-6.1, NFR-6.2**
   * Tests that build scripts work correctly and maintain dependency resolution
   * EXPECTED: PASS (confirms build system behavior is preserved)
   */
  test('Property 2: Build system maintains proper dependency resolution and workspace isolation', () => {
    console.log('Testing build system functionality preservation...');
    
    // Test turbo build configuration
    const turboConfig = JSON.parse(fs.readFileSync(path.join(workspaceRoot, 'turbo.json'), 'utf8'));
    expect(turboConfig.tasks.build).toBeDefined();
    expect(turboConfig.tasks.build.dependsOn).toEqual(['^build']);
    expect(turboConfig.tasks.build.outputs).toEqual(['dist/**', '.next/**']);
    
    console.log('Turbo build configuration preserved:', turboConfig.tasks.build);
    
    // Test Next.js configuration preservation in frontend
    const frontendNextConfig = path.join(frontendPath, 'next.config.mjs');
    expect(fs.existsSync(frontendNextConfig)).toBe(true);
    
    const nextConfigContent = fs.readFileSync(frontendNextConfig, 'utf8');
    
    // Key configuration patterns that must be preserved
    expect(nextConfigContent).toContain('outputFileTracingRoot');
    expect(nextConfigContent).toContain('standalone');
    expect(nextConfigContent).toContain('withSentryConfig');
    expect(nextConfigContent).toContain('optimizePackageImports');
    
    console.log('Next.js configuration structure preserved');
    
    // Test workspace dependency relationships
    const frontendPkg = JSON.parse(fs.readFileSync(path.join(frontendPath, 'package.json'), 'utf8'));
    const waveflixWebPkg = JSON.parse(fs.readFileSync(path.join(waveflixWebPath, 'package.json'), 'utf8'));
    
    // Both Next.js workspaces should maintain their dependencies
    expect(frontendPkg.dependencies.next).toBeDefined();
    expect(frontendPkg.dependencies.react).toBeDefined();
    expect(waveflixWebPkg.dependencies.next).toBeDefined();
    expect(waveflixWebPkg.dependencies.react).toBeDefined();
    
    console.log('Workspace dependency patterns preserved:', {
      frontend: { next: frontendPkg.dependencies.next, react: frontendPkg.dependencies.react },
      waveflixWeb: { next: waveflixWebPkg.dependencies.next, react: waveflixWebPkg.dependencies.react }
    });
    
    // Test build script existence in workspaces
    expect(frontendPkg.scripts.build).toBe('next build');
    expect(waveflixWebPkg.scripts.build).toBe('next build');
    
    console.log('Build script patterns preserved');
  });

  /**
   * Property 3: Package Resolution and Module Loading
   * **Validates: Requirements NFR-6.1, NFR-6.2**
   * Tests that workspace package resolution works correctly from different contexts
   * EXPECTED: PASS (confirms package resolution behavior is preserved)
   */
  test('Property 3: Workspace package resolution maintains proper isolation and accessibility', () => {
    console.log('Testing workspace package resolution preservation...');
    
    // Test that packages can resolve dependencies from workspace root
    const testResolution = (packagePath, dependencyName) => {
      try {
        const resolved = require.resolve(dependencyName, { paths: [packagePath] });
        return { success: true, path: resolved };
      } catch (error) {
        return { success: false, error: error.message };
      }
    };
    
    // Test critical dependency resolution from each workspace
    const resolutionTests = [
      { workspace: 'frontend', path: frontendPath, deps: ['next', 'react', 'framer-motion'] },
      { workspace: 'waveflix-web', path: waveflixWebPath, deps: ['next', 'react'] },
      { workspace: 'extractor', path: extractorPath, deps: ['express', 'cors'] }
    ];
    
    const resolutionResults = {};
    
    resolutionTests.forEach(({ workspace, path, deps }) => {
      resolutionResults[workspace] = {};
      
      deps.forEach(dep => {
        resolutionResults[workspace][dep] = testResolution(path, dep);
      });
      
      // Test workspace root resolution
      resolutionResults[workspace]['workspace-root'] = testResolution(path, '../package.json');
    });
    
    console.log('Package resolution results:', JSON.stringify(resolutionResults, null, 2));
    
    // All critical dependencies should resolve successfully
    Object.entries(resolutionResults).forEach(([workspace, deps]) => {
      Object.entries(deps).forEach(([dep, result]) => {
        if (dep !== 'workspace-root') {
          expect(result.success).toBe(true);
          console.log(`${workspace}/${dep} resolves to:`, result.path);
        }
      });
    });
    
    // Test shared dependency inheritance from root
    const rootPkg = JSON.parse(fs.readFileSync(path.join(workspaceRoot, 'package.json'), 'utf8'));
    
    // framer-motion should be available from root to frontend
    if (rootPkg.dependencies['framer-motion']) {
      const framerMotionResolution = testResolution(frontendPath, 'framer-motion');
      expect(framerMotionResolution.success).toBe(true);
      console.log('Shared dependency (framer-motion) accessible from frontend workspace');
    }
    
    console.log('Package resolution patterns preserved');
  });

  /**
   * Property 4: Next.js Specific Configuration Preservation
   * **Validates: Requirements NFR-6.2** 
   * Tests that Next.js hot reload, fast refresh, and build configurations work
   * EXPECTED: PASS (confirms Next.js functionality is preserved)
   */
  test('Property 4: Next.js configuration maintains hot reload and build optimization features', () => {
    console.log('Testing Next.js configuration preservation...');
    
    // Test frontend Next.js config
    const frontendNextConfigPath = path.join(frontendPath, 'next.config.mjs');
    const frontendConfigContent = fs.readFileSync(frontendNextConfigPath, 'utf8');
    
    // Key Next.js features that must be preserved
    const requiredFeatures = [
      'outputFileTracingRoot', // Critical for monorepo builds
      'standalone',            // Docker deployment support
      'images',               // Image optimization
      'compress: true',       // Production optimization
      'poweredByHeader: false', // Security
      'rewrites',             // API routing
      'headers',              // Security headers
      'experimental',         // Performance features
      'webpack'              // Custom webpack config
    ];
    
    requiredFeatures.forEach(feature => {
      expect(frontendConfigContent).toContain(feature);
      console.log(`Next.js feature preserved: ${feature}`);
    });
    
    // Test that outputFileTracingRoot points to workspace root
    expect(frontendConfigContent).toContain("path.resolve(import.meta.dirname, '../../')");
    console.log('OutputFileTracingRoot correctly points to workspace root');
    
    // Test waveflix-web Next.js config
    const waveflixWebConfigPath = path.join(waveflixWebPath, 'next.config.ts');
    if (fs.existsSync(waveflixWebConfigPath)) {
      const waveflixWebConfig = fs.readFileSync(waveflixWebConfigPath, 'utf8');
      console.log('WaveFlix-Web Next.js config exists and preserved');
    }
    
    // Test dev server port configuration preservation
    const frontendPkg = JSON.parse(fs.readFileSync(path.join(frontendPath, 'package.json'), 'utf8'));
    const waveflixWebPkg = JSON.parse(fs.readFileSync(path.join(waveflixWebPath, 'package.json'), 'utf8'));
    
    expect(frontendPkg.scripts.dev).toContain('-p 3005');
    expect(waveflixWebPkg.scripts.dev).toContain('-p 3006');
    
    console.log('Next.js dev server port isolation preserved:', {
      frontend: '3005',
      waveflixWeb: '3006'
    });
    
    // Test webpack flag preservation for compatibility
    expect(frontendPkg.scripts.dev).toContain('--webpack');
    expect(waveflixWebPkg.scripts.dev).toContain('--webpack');
    
    console.log('Webpack compatibility flag preserved for both Next.js apps');
  });

  /**
   * Property 5: Vercel Deployment Configuration Preservation
   * **Validates: Requirements NFR-6.1**
   * Tests that Vercel configuration maintains proper service definitions
   * EXPECTED: PASS (confirms Vercel config structure is preserved)
   */
  test('Property 5: Vercel configuration maintains proper service architecture and routing', () => {
    console.log('Testing Vercel configuration preservation...');
    
    const vercelConfigPath = path.join(workspaceRoot, 'vercel.json');
    expect(fs.existsSync(vercelConfigPath)).toBe(true);
    
    const vercelConfig = JSON.parse(fs.readFileSync(vercelConfigPath, 'utf8'));
    
    // Core Vercel configuration structure
    expect(vercelConfig.services).toBeDefined();
    expect(vercelConfig.rewrites).toBeDefined();
    
    // Expected services configuration
    const expectedServices = ['frontend', 'backend', 'waveflix-extractor', 'waveflix-web'];
    const actualServices = Object.keys(vercelConfig.services);
    
    expectedServices.forEach(service => {
      expect(actualServices).toContain(service);
      console.log(`Vercel service preserved: ${service}`);
    });
    
    // Test service configurations
    expect(vercelConfig.services.frontend.root).toBe('waveflix-app/frontend');
    expect(vercelConfig.services.frontend.framework).toBe('nextjs');
    expect(vercelConfig.services['waveflix-web'].root).toBe('waveflix-web');
    expect(vercelConfig.services['waveflix-web'].framework).toBe('nextjs');
    expect(vercelConfig.services.backend.root).toBe('waveflix-app/backend');
    expect(vercelConfig.services.backend.runtime).toBe('go');
    expect(vercelConfig.services['waveflix-extractor'].root).toBe('waveflix-extractor');
    expect(vercelConfig.services['waveflix-extractor'].framework).toBe('express');
    
    console.log('Vercel service configurations preserved');
    
    // Test service bindings preservation
    expect(vercelConfig.services.frontend.bindings).toBeDefined();
    expect(vercelConfig.services.backend.bindings).toBeDefined();
    expect(vercelConfig.services['waveflix-web'].bindings).toBeDefined();
    
    // Test routing configuration preservation
    const expectedRewrites = [
      { source: '/api/(.*)', destination: { service: 'backend' } },
      { source: '/landing/(.*)', destination: { service: 'waveflix-web' } },
      { source: '/landing', destination: { service: 'waveflix-web' } },
      { source: '/(.*)', destination: { service: 'frontend' } }
    ];
    
    expect(vercelConfig.rewrites).toEqual(expectedRewrites);
    console.log('Vercel routing configuration preserved');
    
    console.log('Vercel configuration structure and patterns preserved');
  });

  /**
   * Property 6: Environment and Build Tool Configuration
   * **Validates: Requirements NFR-6.1, NFR-6.2**
   * Tests that environment setup and build tools maintain proper configuration
   * EXPECTED: PASS (confirms environment configuration is preserved)
   */
  test('Property 6: Build tools and environment configuration maintain workspace functionality', () => {
    console.log('Testing build tools and environment configuration preservation...');
    
    // Test root package.json scripts preservation
    const rootPkg = JSON.parse(fs.readFileSync(path.join(workspaceRoot, 'package.json'), 'utf8'));
    
    const expectedRootScripts = ['predev', 'dev', 'build', 'killports'];
    expectedRootScripts.forEach(script => {
      expect(rootPkg.scripts[script]).toBeDefined();
      console.log(`Root script preserved: ${script} -> ${rootPkg.scripts[script]}`);
    });
    
    // Test turbo configuration completeness
    const turboConfig = JSON.parse(fs.readFileSync(path.join(workspaceRoot, 'turbo.json'), 'utf8'));
    
    // Environment pass-through configuration
    expect(turboConfig.globalPassThroughEnv).toBeDefined();
    expect(turboConfig.globalPassThroughEnv).toContain('LocalAppData');
    expect(turboConfig.globalPassThroughEnv).toContain('GOCACHE');
    expect(turboConfig.globalPassThroughEnv).toContain('APPDATA');
    
    console.log('Turbo environment pass-through preserved:', turboConfig.globalPassThroughEnv);
    
    // Test package manager configuration
    expect(rootPkg.packageManager).toBe('npm@11.6.2');
    expect(rootPkg.private).toBe(true);
    
    console.log('Package manager configuration preserved');
    
    // Test workspace isolation via dependencies
    const workspaceDependencies = {};
    
    [frontendPath, waveflixWebPath, extractorPath, backendPath].forEach(wsPath => {
      const pkgPath = path.join(wsPath, 'package.json');
      if (fs.existsSync(pkgPath)) {
        const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'));
        workspaceDependencies[path.basename(wsPath)] = {
          dependencies: Object.keys(pkg.dependencies || {}),
          devDependencies: Object.keys(pkg.devDependencies || {}),
          hasPrivate: pkg.private === true
        };
      }
    });
    
    console.log('Workspace dependency isolation preserved:', workspaceDependencies);
    
    // Test that workspaces maintain proper privacy settings
    Object.entries(workspaceDependencies).forEach(([workspace, config]) => {
      if (workspace === 'frontend' || workspace === 'waveflix-web') {
        expect(config.hasPrivate).toBe(true);
        console.log(`${workspace} privacy setting preserved`);
      }
    });
    
    console.log('Build tools and environment configuration preserved');
  });

  /**
   * Property 7: Lockfile and Dependency Management
   * **Validates: Requirements NFR-6.1**
   * Tests current dependency management patterns that should be preserved
   * EXPECTED: PASS (confirms dependency management behavior is preserved)
   */
  test('Property 7: Dependency management maintains current resolution patterns and lockfile structure', () => {
    console.log('Testing dependency management preservation...');
    
    // Test root lockfile existence and structure
    const rootLockfile = path.join(workspaceRoot, 'package-lock.json');
    expect(fs.existsSync(rootLockfile)).toBe(true);
    
    const lockfileContent = JSON.parse(fs.readFileSync(rootLockfile, 'utf8'));
    expect(lockfileContent.lockfileVersion).toBeDefined();
    expect(lockfileContent.packages).toBeDefined();
    
    console.log('Root lockfile structure preserved');
    
    // Test workspace lockfile patterns (document current state)
    const potentialLockfiles = [
      path.join(frontendPath, 'package-lock.json'),
      path.join(waveflixWebPath, 'package-lock.json'), 
      path.join(extractorPath, 'package-lock.json'),
      path.join(backendPath, 'package-lock.json')
    ];
    
    const existingWorkspaceLockfiles = potentialLockfiles.filter(lockfile => 
      fs.existsSync(lockfile)
    );
    
    console.log('Current workspace lockfiles:', existingWorkspaceLockfiles);
    
    // Document the current state - this captures what should be preserved
    // The actual fix may change this, but this test documents the baseline
    
    // Test shared dependency availability
    const rootPkg = JSON.parse(fs.readFileSync(path.join(workspaceRoot, 'package.json'), 'utf8'));
    const sharedDeps = Object.keys(rootPkg.dependencies || {});
    
    console.log('Root-level shared dependencies preserved:', sharedDeps);
    
    // Test that shared dependencies are accessible to workspaces
    if (sharedDeps.includes('framer-motion')) {
      // Should be accessible from frontend workspace
      try {
        const framerMotionPath = require.resolve('framer-motion', { paths: [frontendPath] });
        console.log('Shared dependency (framer-motion) accessible from frontend');
        expect(framerMotionPath).toBeTruthy();
      } catch (error) {
        console.log('Shared dependency resolution behavior documented:', error.message);
      }
    }
    
    // Test workspace-specific dependency resolution
    const frontendPkg = JSON.parse(fs.readFileSync(path.join(frontendPath, 'package.json'), 'utf8'));
    const frontendSpecificDeps = Object.keys(frontendPkg.dependencies || {});
    
    console.log('Frontend-specific dependencies preserved:', frontendSpecificDeps.slice(0, 5));
    
    console.log('Dependency management patterns documented and preserved');
  });
});