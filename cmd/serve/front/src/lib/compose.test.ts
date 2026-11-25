import { describe, expect, test } from 'vitest';
import { composeProblem, composeServices } from './compose.ts';
import { templates } from './templates/catalog.ts';

const file = `services:
  # the API
  api:
    image: team/api:1
    ports:
      - "8080:8080"
  db:
    image: postgres:18.7-alpine

volumes:
  data:
`;

describe('composeProblem', () => {
	test('accepts a compose file with services', () => {
		expect(composeProblem(file)).toBeUndefined();
		for (const template of templates) expect(composeProblem(template.compose)).toBeUndefined();
	});

	test('says what stops a file first', () => {
		expect(composeProblem('  \n')).toBe('compose.problem.empty');
		expect(composeProblem('services:\n\tapi:\n\t\timage: x\n')).toBe('compose.problem.tabs');
		expect(composeProblem('version: "3"\nvolumes:\n  data:\n')).toBe('compose.problem.no_services');
		expect(composeProblem('  services:\n    api:\n')).toBe('compose.problem.no_services');
	});
});

describe('composeServices', () => {
	test('lists the top services only', () => {
		expect(composeServices(file)).toEqual(['api', 'db']);
	});

	test('is empty without services', () => {
		expect(composeServices('volumes:\n  data:\n')).toEqual([]);
	});
});
