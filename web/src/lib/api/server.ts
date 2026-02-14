import createClient from 'openapi-fetch';
import type { paths } from './types';

const BFF_URL = process.env.BFF_URL || 'http://localhost:8080';

export function createServerClient(accessToken?: string) {
	const headers: Record<string, string> = {};
	if (accessToken) {
		headers['Authorization'] = `Bearer ${accessToken}`;
	}

	return createClient<paths>({
		baseUrl: BFF_URL,
		headers
	});
}

export { BFF_URL };
