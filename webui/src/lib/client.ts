class Client {
	public address: string;
	public secret: string;
	public onUpdate?: () => void;

	constructor(address: string, secret: string) {
		this.address = address;
		this.secret = secret;
	}

	private async createHmac(data: string): Promise<string> {
		const encoder = new TextEncoder();
		const keyData = encoder.encode(this.secret);
		const key = await crypto.subtle.importKey(
			'raw',
			keyData,
			{ name: 'HMAC', hash: 'SHA-256' },
			false,
			['sign']
		);
		const signature = await crypto.subtle.sign('HMAC', key, encoder.encode(data));
		return Array.from(new Uint8Array(signature))
			.map((b) => b.toString(16).padStart(2, '0'))
			.join('');
	}

	private async fetch(method: string, path: string, body?: unknown): Promise<Response> {
		const url = `${this.address}/${path}`;
		const headers: HeadersInit = {
			'Content-Type': 'application/json'
		};

		const stringBody = body ? JSON.stringify(body) : '';

		const hmac = await this.createHmac(stringBody);
		headers['X-Signature-256'] = `sha256=${hmac}`;

		const response = await fetch(url, {
			method,
			headers,
			body: stringBody || undefined
		});

		if (!response.ok) {
			const responseBody = await response.text();
			throw new Error(`Unexpected status code: ${response.status}, body: ${responseBody}`);
		}

		return response;
	}

	async getConfig(): Promise<Config> {
		const response = await this.fetch('GET', 'api/config');
		return response.json();
	}

	async services(): Promise<Service[]> {
		const response = await this.fetch('GET', 'api/services');
		return response.json();
	}

	async service(name: string): Promise<Service> {
		const response = await this.fetch('GET', `api/services/${name}`);
		return response.json();
	}

	async startService(name: string): Promise<void> {
		await this.fetch('GET', `api/services/${name}/start`);
		this.onUpdate?.();
	}

	async stopService(name: string): Promise<void> {
		await this.fetch('GET', `api/services/${name}/stop`);
		this.onUpdate?.();
	}

	async updateService(name: string): Promise<void> {
		await this.fetch('GET', `api/services/${name}/update`);
		this.onUpdate?.();
	}

	async createService(config: ServiceConfig): Promise<void> {
		await this.fetch('POST', 'api/services', config);
		this.onUpdate?.();
	}

	async deleteService(name: string): Promise<void> {
		await this.fetch('DELETE', `api/services/${name}`);
		this.onUpdate?.();
	}

	async restartService(name: string): Promise<void> {
		await this.fetch('GET', `api/services/${name}/restart`);
		this.onUpdate?.();
	}
}

interface ProxyConfig {
	match: string;
	upstream: string;
}

interface Config {
	loadPath?: string;
	services: { [key: string]: ServiceConfig };
	address: string;
	servicesPath: string;
	secret: string;
}

interface Service {
	config: ServiceConfig;
	path: string;
	status: ServiceStatus;
	restarts: number;
	logs: string[];
}

interface ServiceConfig {
	name: string;
	repo: string;
	exec: string;
	build: string;
	restart: boolean;
	maxRestarts: number;
	secret: string;
	proxy: ProxyConfig;
}

enum ServiceStatus {
	Running = 0,
	Stopped = 1
}

export type { ProxyConfig, Config, Service, ServiceConfig };

export { Client, ServiceStatus };
