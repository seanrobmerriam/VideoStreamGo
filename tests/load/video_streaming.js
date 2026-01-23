// k6 load test for video streaming
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 20 },
    { duration: '2m', target: 100 },
    { duration: '30s', target: 0 },
  ],
};

// Video streaming metrics
export default function () {
  const baseUrl = 'http://localhost:8080';

  // Get video list
  const videos = http.get(`${baseUrl}/api/videos`);
  check(videos, { 'get videos': (r) => r.status === 200 });

  // Get video details
  const videoDetail = http.get(`${baseUrl}/api/videos/video-123`);
  check(videoDetail, { 'video detail': (r) => r.status === 200 });

  // Get streaming manifest
  const manifest = http.get(`${baseUrl}/api/videos/video-123/manifest.m3u8`);
  check(manifest, { 'streaming manifest': (r) => r.status === 200 });

  sleep(2);
}
