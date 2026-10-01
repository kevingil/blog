#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DormantCapabilityKind {
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
        name: "nested_agent",
        kind: DormantCapabilityKind::AgentTool,
        disposition: "inventory-only: not registered by the blog copilot manager",
    },
];
