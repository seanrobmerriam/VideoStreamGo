// k6 load test for video upload
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 5 },
    { duration: '1m', target: 20 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<2000'],
  },
};

// Video upload tests
export default function () {
  const baseUrl = 'http://localhost:8080';

  // Initiate upload
  const uploadInit = http.post(`${baseUrl}/api/videos/upload/init`, '{}', {
    headers: { 'Authorization': 'Bearer test-token' },
  });
  check(uploadInit, { 'init upload': (r) => r.status === 200 });

  // Upload chunk (simulated)
  const uploadChunk = http.post(`${baseUrl}/api/videos/upload/chunk`, 'chunk-data', {
    headers: { 
      'Authorization': 'Bearer test-token',
      'Content-Type': 'application/octet-stream',
    },
  });
  check(uploadChunk, { 'upload chunk': (r) => r.status === 200 });

  // Complete upload
  const uploadComplete = http.post(`${baseUrl}/api/videos/upload/complete`, '{}', {
    headers: { 'Authorization': 'Bearer test-token' },
  });
  check(uploadComplete, { 'complete upload': (r) => r.status === 200 });

  sleep(3);
}
