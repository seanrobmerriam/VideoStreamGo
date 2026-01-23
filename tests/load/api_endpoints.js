// k6 load test for API endpoints
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 10 },
    { duration: '1m', target: 50 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.01'],
  },
};

// Test API endpoints
export default function () {
  const baseUrl = 'http://localhost:8080';

  // Health check
  const health = http.get(`${baseUrl}/health`);
  check(health, { 'health check': (r) => r.status === 200 });

  // List customers
  const customers = http.get(`${baseUrl}/api/customers`, {
    headers: { 'Authorization': 'Bearer test-token' },
  });
  check(customers, { 'list customers': (r) => r.status === 200 });

  // List instances
  const instances = http.get(`${baseUrl}/api/instances`, {
    headers: { 'Authorization': 'Bearer test-token' },
  });
  check(instances, { 'list instances': (r) => r.status === 200 });

  sleep(1);
}
