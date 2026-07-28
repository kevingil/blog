#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DormantCapabilityKind {
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
/// root. Provider adapters are implemented and contract-tested separately;
/// only capabilities with no active registration remain in this inventory.
pub const DORMANT_CAPABILITIES: &[DormantCapability] = &[
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
