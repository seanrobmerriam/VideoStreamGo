// k6 load test for tenant isolation
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 10 },
    { duration: '1m', target: 30 },
    { duration: '30s', target: 0 },
  ],
};

// Tenant isolation verification
export default function () {
  const baseUrl = 'http://localhost:8080';

  // Tenant 1 data access
  const tenant1Data = http.get(`${baseUrl}/api/data`, {
    headers: { 'Authorization': 'Bearer tenant_1_token' },
  });
  check(tenant1Data, { 'tenant 1 data': (r) => r.status === 200 });

  // Tenant 2 data access
  const tenant2Data = http.get(`${baseUrl}/api/data`, {
    headers: { 'Authorization': 'Bearer tenant_2_token' },
  });
  check(tenant2Data, { 'tenant 2 data': (r) => r.status === 200 });

  // Cross-tenant access should be denied
  const crossTenant = http.get(`${baseUrl}/api/tenant/1/data`, {
    headers: { 'Authorization': 'Bearer tenant_2_token' },
  });
  check(crossTenant, { 'cross tenant denied': (r) => r.status === 403 });

  sleep(1);
}
