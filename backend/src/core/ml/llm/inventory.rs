#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DormantCapabilityKind {
    Provider,
    McpTransport,
    AgentTool,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct DormantCapability {
    pub name: &'static str,
    pub kind: DormantCapabilityKind,
    pub disposition: &'static str,
}

/// These Go paths exist but are not configured by the active blog composition
/// root. They are recorded explicitly so a future product decision can port
/// them without pretending that unverified behavior is live today.
pub const DORMANT_CAPABILITIES: &[DormantCapability] = &[
    DormantCapability {
        name: "anthropic",
        kind: DormantCapabilityKind::Provider,
        disposition: "inventory-only: disabled by default in Go",
    },
    DormantCapability {
        name: "gemini",
        kind: DormantCapabilityKind::Provider,
        disposition: "inventory-only: disabled by default in Go",
    },
    DormantCapability {
        name: "groq",
        kind: DormantCapabilityKind::Provider,
        disposition: "inventory-only: disabled by default in Go",
    },
    DormantCapability {
        name: "vertex_ai",
        kind: DormantCapabilityKind::Provider,
        disposition: "inventory-only: no active blog configuration",
    },
    DormantCapability {
        name: "mcp_stdio",
        kind: DormantCapabilityKind::McpTransport,
        disposition: "inventory-only: MCP server map is empty in active Go configuration",
    },
    DormantCapability {
        name: "mcp_sse",
        kind: DormantCapabilityKind::McpTransport,
        disposition: "inventory-only: MCP server map is empty in active Go configuration",
    },
    DormantCapability {
        name: "nested_agent",
        kind: DormantCapabilityKind::AgentTool,
        disposition: "inventory-only: not registered by the blog copilot manager",
    },
];
