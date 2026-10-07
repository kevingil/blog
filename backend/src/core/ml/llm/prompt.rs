use std::collections::BTreeSet;

/// Context for the blog copilot. Tool schemas travel with the request.
/// This only names the function tools registered for the turn, so the model
/// calls those instead of inventing others. Hosted tools (`web_search`,
/// `sandbox`) are omitted here; the provider runs them and they are not functions.
pub fn copilot_prompt(available_tools: &[String]) -> String {
    let tools = available_tools
        .iter()
        .map(String::as_str)
        .filter(|name| !is_hosted_tool(name))
        .collect::<BTreeSet<_>>();

    let mut prompt = String::from(
        "\
You are the author's ghostwriter. Research with your tools and write in their voice.

If they told you to edit, edit. If you are about to change the article and they did not ask, confirm first. If you are unsure, ask.",
    );
    if !tools.is_empty() {
        let names = tools.into_iter().collect::<Vec<_>>().join(", ");
        prompt.push_str("\n\nTools: ");
        prompt.push_str(&names);
        prompt.push('.');
    }
    prompt
}

fn is_hosted_tool(name: &str) -> bool {
    matches!(name, "web_search" | "sandbox" | "code_interpreter")
}
