export interface LLMModel {
    id: string;
    name: string;
}

export const fetchAvailableModels = async (
    provider: string,
    apiKeyOrHost: string,
    fallbackModels: string[]
): Promise<string[]> => {
    if (!apiKeyOrHost) {
        return fallbackModels;
    }

    try {
        switch (provider) {
            case 'gemini':
                return await fetchGeminiModels(apiKeyOrHost);
            case 'openai':
                return await fetchOpenAIModels(apiKeyOrHost);
            case 'deepseek':
                return await fetchDeepSeekModels(apiKeyOrHost);
            case 'ollama':
                return await fetchOllamaModels(apiKeyOrHost);
            case 'anthropic':
                // Anthropic does not provide a public model listing endpoint
                return fallbackModels;
            default:
                return fallbackModels;
        }
    } catch (error) {
        console.error(`Failed to fetch dynamic models for ${provider}:`, error);
        return fallbackModels; // Fallback to hardcoded list on error
    }
};

const fetchGeminiModels = async (apiKey: string): Promise<string[]> => {
    const res = await fetch(`https://generativelanguage.googleapis.com/v1beta/models?key=${apiKey}`);
    if (!res.ok) throw new Error('API request failed');
    const data = await res.json();
    return data.models
        .filter((model: any) => model.supportedGenerationMethods.includes('generateContent'))
        .map((model: any) => model.name.replace('models/', ''));
};

const fetchOpenAIModels = async (apiKey: string): Promise<string[]> => {
    const res = await fetch('https://api.openai.com/v1/models', {
        headers: {
            'Authorization': `Bearer ${apiKey}`
        }
    });
    if (!res.ok) throw new Error('API request failed');
    const data = await res.json();
    // OpenAI returns many models (whisper, dall-e, etc). We ideally only want chat models.
    return data.data
        .map((model: any) => model.id)
        .filter((id: string) => id.startsWith('gpt') || id.startsWith('o1') || id.startsWith('o3'))
        .sort();
};

const fetchDeepSeekModels = async (apiKey: string): Promise<string[]> => {
    const res = await fetch('https://api.deepseek.com/models', {
        headers: {
            'Authorization': `Bearer ${apiKey}`
        }
    });
    if (!res.ok) throw new Error('API request failed');
    const data = await res.json();
    return data.data.map((model: any) => model.id).sort();
};

const fetchOllamaModels = async (host: string): Promise<string[]> => {
    // Attempt to hit the Ollama tags endpoint. May fail due to CORS.
    const url = host.endsWith('/') ? `${host}api/tags` : `${host}/api/tags`;
    const res = await fetch(url);
    if (!res.ok) throw new Error('API request failed');
    const data = await res.json();
    return data.models.map((model: any) => model.name).sort();
};
