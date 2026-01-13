import api from './api';
import type { User } from '../types';

export const userService = {
    updateLLMConfig: async (config: Record<string, any>) => {
        const response = await api.patch<{ success: boolean; data: User }>('/auth/me/llm-config', config);
        return response.data;
    }
};
