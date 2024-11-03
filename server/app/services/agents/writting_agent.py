from typing import Dict, List, Optional
from langchain_anthropic import ChatAnthropic
from langchain.tools import TavilySearchResults
from langgraph.graph import END, StateGraph

class BlogState:
    def __init__(self):
        self.messages: List[dict] = []
        self.topic: str = ""
        self.research_data: List[dict] = []
        self.outline: List[str] = []
        self.current_section: Optional[str] = None
        self.draft: str = ""
        self.status: str = "initialized"
        self.metadata: Dict = {
            "suggested_tags": [],
            "estimated_read_time": 0,
            "seo_keywords": []
        }
        
    def dict(self) -> dict:
        return {
            "messages": self.messages,
            "topic": self.topic,
            "research_data": self.research_data,
            "outline": self.outline,
            "current_section": self.current_section,
            "draft": self.draft,
            "status": self.status,
            "metadata": self.metadata
        }

class BlogWriterAgent:
    def __init__(self):
        self.llm = ChatAnthropic(model="claude-3-opus-20240229")
        self.search_tool = TavilySearchResults()
        
    def research_topic(self, state: Dict) -> Dict:
        """Research the blog topic using Tavily."""
        topic = state["topic"]
        search_results = self.search_tool.invoke(f"latest information about {topic}")
        
        state["research_data"] = search_results
        return state

    def create_outline(self, state: Dict) -> Dict:
        research_context = "\n".join([
            f"Source {i+1}: {result['title']} - {result['content'][:200]}..."
            for i, result in enumerate(state["research_data"])
        ])
        
        prompt = f"""Based on the following research about {state['topic']}, create a detailed blog post outline:

        Research:
        {research_context}

        Create an outline with main sections and subsections that covers the topic comprehensively.
        """
        
        response = self.llm.invoke(prompt)
        outline = [line.strip() for line in response.content.split('\n') if line.strip()]
        
        state["outline"] = outline
        return state

    def write_section(self, state: Dict) -> Dict:
        if not state["current_section"]:
            state["current_section"] = state["outline"][0]
        
        research_context = "\n".join([
            f"Source {i+1}: {result['title']} - {result['content']}"
            for i, result in enumerate(state["research_data"])
        ])
        
        prompt = f"""Write the following section of a blog post about {state['topic']}:

        Section: {state['current_section']}

        Use this research for accurate information:
        {research_context}

        Write in an engaging, conversational style. Include relevant examples and data from the research.
        """
        
        response = self.llm.invoke(prompt)
        
        if state["draft"]:
            state["draft"] += "\n\n"
        state["draft"] += f"## {state['current_section']}\n\n{response.content}"
        
        current_index = state["outline"].index(state["current_section"])
        if current_index + 1 < len(state["outline"]):
            state["current_section"] = state["outline"][current_index + 1]
        else:
            state["current_section"] = None
            
        return state

    def analyze_content(self, state: Dict) -> Dict:
        """Analyze the content to extract metadata."""
        prompt = f"""Analyze this blog post and provide:
        1. 5-7 relevant tags
        2. Estimated read time in minutes
        3. 3-5 main SEO keywords

        Content:
        {state['draft']}
        """
        
        response = self.llm.invoke(prompt)
        # Parse the response to update metadata
        
        state["metadata"]["suggested_tags"] = ["AI", "Technology", "Tutorial"]  # Example
        state["metadata"]["estimated_read_time"] = len(state["draft"].split()) // 200  # Rough estimate
        state["metadata"]["seo_keywords"] = ["artificial intelligence", "machine learning"]  # Example
        
        return state

    def create_workflow(self) -> StateGraph:
        workflow = StateGraph()
        
        # Add nodes
        workflow.add_node("research", self.research_topic)
        workflow.add_node("create_outline", self.create_outline)
        workflow.add_node("write_section", self.write_section)
        workflow.add_node("analyze_content", self.analyze_content)
        
        # Add edges
        workflow.add_edge("research", "create_outline")
        workflow.add_edge("create_outline", "write_section")
        workflow.add_edge("write_section", "should_continue")
        
        def should_continue(state: Dict) -> str:
            if state["current_section"] is None and state["draft"]:
                return "analyze"
            return "continue_writing"
        
        workflow.add_conditional_edges(
            "should_continue",
            should_continue,
            {
                "continue_writing": "write_section",
                "analyze": "analyze_content"
            }
        )
        
        workflow.add_edge("analyze_content", END)
        
        workflow.set_entry_point("research")
        return workflow
