use std::sync::{Arc, RwLock};

use super::Tool;

/// Built-in copilot tools plus dynamically attached MCP connector tools.
pub struct ToolRegistry {
    builtin: Vec<Arc<dyn Tool>>,
    extra: RwLock<Vec<Arc<dyn Tool>>>,
}

impl ToolRegistry {
    pub fn from_builtin(tools: Vec<Arc<dyn Tool>>) -> Arc<Self> {
        Arc::new(Self {
            builtin: tools,
            extra: RwLock::new(Vec::new()),
        })
    }

    pub fn snapshot(&self) -> Vec<Arc<dyn Tool>> {
        let extra = self.extra.read().map(|tools| tools.clone()).unwrap_or_default();
        let mut tools = self.builtin.clone();
        tools.extend(extra);
        tools
    }

    pub fn replace_extra(&self, tools: Vec<Arc<dyn Tool>>) {
        if let Ok(mut extra) = self.extra.write() {
            *extra = tools;
        }
    }

    pub fn names(&self) -> Vec<String> {
        self.snapshot()
            .into_iter()
            .map(|tool| tool.info().name)
            .collect()
    }

    pub fn describe(&self) -> Vec<RegisteredTool> {
        self.snapshot()
            .into_iter()
            .map(|tool| {
                let info = tool.info();
                let source = if info.name.contains("__") {
                    "mcp".to_owned()
                } else {
                    "builtin".to_owned()
                };
                RegisteredTool {
                    name: info.name,
                    description: info.description,
                    source,
                }
            })
            .collect()
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RegisteredTool {
    pub name: String,
    pub description: String,
    pub source: String,
}
