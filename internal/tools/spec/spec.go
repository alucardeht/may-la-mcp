package spec

// Package spec implements spec-driven development tooling for May-la projects.
//
// It provides a complete framework for managing specifications, constitutions,
// implementation plans, and task tracking. The module integrates seamlessly with
// the May-la CLI through the tool registry system.
//
// Core Components:
//   - InitTool: Initializes .mayla/ project structure
//   - GenerateTool: Generates specification artifacts from templates
//   - ValidateTool: Validates consistency across artifacts
//   - StatusTool: Reports project specification status
//   - Constitution: Parses and validates project constitution documents
//
// Directory Structure:
//   .mayla/
//   ├── constitution.md   (project principles, constraints, governance)
//   ├── spec.md          (feature specifications with user stories)
//   ├── plan.md          (implementation plan with phases)
//   ├── tasks.md         (detailed task breakdown)
//   ├── memories/        (context and decisions cache)
//   └── cache/           (temporary build artifacts)
//
// Usage:
//   registry := GetTools()
//   result, err := registry.Execute("spec_init", map[string]interface{}{
//     "path": "/project/path",
//     "force": false,
//   })
