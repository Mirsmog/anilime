import http from 'k6/http';
import { check, sleep } from 'k6';

// Accept 200, 201, 202, 204 as successful responses (202 = async writes, 204 = deletes)
http.setResponseCallback(http.expectedStatuses(200, 201, 202, 204));

const BASE_URL = 'http://localhost:8080';

export const options = {
  stages: [
    { duration: '30s', target: 50 },
    { duration: '1m', target: 200 },
    { duration: '2m', target: 500 },
    { duration: '1m', target: 1000 },
    { duration: '2m', target: 1000 },
    { duration: '1m', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.1'],
  },
};

export function setup() {
  const email = 'load_shared@example.com';
  const password = 'Test1234!';
  
  let loginRes = http.post(`${BASE_URL}/v1/auth/login`, JSON.stringify({
    login: email, password
  }), { headers: { 'Content-Type': 'application/json' } });
  
  if (loginRes.status !== 200) {
    http.post(`${BASE_URL}/v1/auth/register`, JSON.stringify({
      email, username: 'load_shared', password
    }), { headers: { 'Content-Type': 'application/json' } });
    
    loginRes = http.post(`${BASE_URL}/v1/auth/login`, JSON.stringify({
      login: email, password
    }), { headers: { 'Content-Type': 'application/json' } });
  }
  
  return { token: loginRes.json('access_token') };
}

export default function (data) {
  const authHeaders = { 'Authorization': `Bearer ${data.token}`, 'Content-Type': 'application/json' };

  // Search
  const searchRes = http.get(`${BASE_URL}/v1/search?q=anime&limit=5`);
  check(searchRes, { 'search ok': (r) => r.status === 200 });

  // Catalog list
  const animeListRes = http.get(`${BASE_URL}/v1/anime?limit=10&offset=0`);
  check(animeListRes, { 'anime list ok': (r) => r.status === 200 });

  const animeList = animeListRes.json('anime') || [];
  if (animeList.length > 0) {
    const animeId = animeList[0].id;

    // Anime details
    const animeRes = http.get(`${BASE_URL}/v1/anime/${animeId}`);
    check(animeRes, { 'anime details ok': (r) => r.status === 200 });

    // Episodes
    const episodesRes = http.get(`${BASE_URL}/v1/anime/${animeId}/episodes?limit=5`);
    check(episodesRes, { 'episodes ok': (r) => r.status === 200 });

    const episodes = episodesRes.json('episodes') || [];
    if (episodes.length > 0) {
      const episodeId = episodes[0].id;

      // Activity progress
      const progressRes = http.post(`${BASE_URL}/v1/activity/progress`, JSON.stringify({
        episode_id: episodeId,
        position_seconds: 120,
        duration_seconds: 1440
      }), { headers: authHeaders });
      check(progressRes, { 'progress ok': (r) => r.status === 200 || r.status === 202 });

      // Comments
      const commentRes = http.post(`${BASE_URL}/v1/comments/${animeId}`, JSON.stringify({
        body: `Load test comment ${__VU}_${__ITER}`
      }), { headers: authHeaders });
      check(commentRes, { 'comment ok': (r) => r.status === 200 || r.status === 201 || r.status === 202 });
    }
  }

  // Continue watching
  http.get(`${BASE_URL}/v1/activity/continue?limit=5`, { headers: authHeaders });

  // Comments list
  if (animeList.length > 0) {
    http.get(`${BASE_URL}/v1/comments/${animeList[0].id}?sort=new&limit=5`);
  }

  sleep(1);
}
