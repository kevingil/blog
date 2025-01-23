'use client'

import { CopilotKit } from '@copilotkit/react-core';
import "@copilotkit/react-textarea/styles.css";
import { CopilotTextarea } from '@copilotkit/react-textarea';

export default function AITextArea({ article, setValue }: { article: any, setValue: any }) {
    return (
        <CopilotKit runtimeUrl="/api/copilotkit">
            <CopilotTextarea
                className="w-full p-4 border border-gray-300 rounded-md"
                value={article?.content || ''}
                onValueChange={(value) => {
                setValue('content', value);
            }}
            autosuggestionsConfig={{
                textareaPurpose: "the body of a blog post",
                chatApiConfigs: {},
            }}
        />
        </CopilotKit>
    );
}
