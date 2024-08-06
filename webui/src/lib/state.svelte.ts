import { type Service, Client } from './client';

export const config: {
	address: string;
	secret: string;
} = $state({
	address: import.meta.env.DEV ? 'http://localhost:1234' : window.location.origin,
	secret: 'secret'
});

export const getClient = () => client;

export const client = new Client(config.address, config.secret);
client.onUpdate = () => {
	loadServices();
};

export const services: Service[] = $state([]);
export const loadServices = async () => {
	const res = await client.services();
	services.splice(0, services.length, ...res);
	services.sort((a, b) => a.config.name.localeCompare(b.config.name));
};
