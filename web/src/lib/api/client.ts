import createClient from 'openapi-fetch';
import type { paths } from './types';

const BFF_URL = import.meta.env.VITE_BFF_URL || 'http://localhost:8080';

export const api = createClient<paths>({
	baseUrl: BFF_URL
});

export function createAuthenticatedClient(accessToken: string) {
	return createClient<paths>({
		baseUrl: BFF_URL,
		headers: {
			Authorization: `Bearer ${accessToken}`
		}
	});
}

export type { paths, components } from './types';
