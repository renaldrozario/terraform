package terraform

import (
	"github.com/hashicorp/terraform/config/module"
)

// ImportGraphBuilder implements GraphBuilder and is responsible for building
// a graph for importing resources into Terraform. This is a much, much
// simpler graph than a normal configuration graph.
type ImportGraphBuilder struct {
	// ImportTargets are the list of resources to import.
	ImportTargets []*ImportTarget

	// Module is the module to add to the graph. See ImportOpts.Module.
	Module *module.Tree

	// Providers is the list of providers supported.
	Providers []string
}

// Build builds the graph according to the steps returned by Steps.
func (b *ImportGraphBuilder) Build(path []string) (*Graph, error) {
	return (&BasicGraphBuilder{
		Steps:    b.Steps(),
		Validate: true,
		Name:     "ImportGraphBuilder",
	}).Build(path)
}

// Steps returns the ordered list of GraphTransformers that must be executed
// to build a complete graph.
func (b *ImportGraphBuilder) Steps() []GraphTransformer {
	// Get the module. If we don't have one, we just use an empty tree
	// so that the transform still works but does nothing.
	mod := b.Module
	if mod == nil {
		mod = module.NewEmptyTree()
	}

	// Custom factory for creating providers.
	providerFactory := func(name string, path []string) GraphNodeProvider {
		return &NodeApplyableProvider{
			NameValue: name,
			PathValue: path,
		}
	}

	steps := []GraphTransformer{
		// Create all our resources from the configuration and state
		&ConfigTransformerOld{Module: mod},

		// Add the import steps
		&ImportStateTransformer{Targets: b.ImportTargets},

		// Provider-related transformations
		&MissingProviderTransformer{Providers: b.Providers, Factory: providerFactory},
		&ProviderTransformer{},
		&DisableProviderTransformerOld{},
		&PruneProviderTransformer{},
		&AttachProviderConfigTransformer{Module: mod},

		// This validates that the providers only depend on variables
		&ImportProviderValidateTransformer{},

		// Single root
		&RootTransformer{},

		// Optimize
		&TransitiveReductionTransformer{},
	}

	return steps
}
