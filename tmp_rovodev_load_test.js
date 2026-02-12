import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Метрики
const errorRate = new Rate('errors');

// Конфигурация нагрузки
export const options = {
  stages: [
    { duration: '30s', target: 50 },   // Разогрев: 0 → 50 users
    { duration: '1m', target: 200 },   // Рост: 50 → 200 users
    { duration: '2m', target: 500 },   // Пик: 200 → 500 users
    { duration: '1m', target: 1000 },  // Максимум: 500 → 1000 users
    { duration: '2m', target: 1000 },  // Удержание: 1000 users
    { duration: '1m', target: 0 },     // Спад: 1000 → 0
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% запросов < 500ms
    errors: ['rate<0.1'],              // Ошибок < 10%
  },
};

const BASE_URL = 'http://localhost:8080';

// Регистрация/логин выполняются один раз per VU
export function setup() {
  // Создаём тестового админа для backfill (если нужно)
  return { baseUrl: BASE_URL };
}

export default function (data) {
  const userId = `user_${__VU}_${__ITER}`;
  const email = `${userId}@loadtest.local`;
  const password = 'LoadTest1234!';

  // 1. Регистрация (или используем существующий токен)
  let token = '';
  const registerRes = http.post(`${BASE_URL}/v1/auth/register`, JSON.stringify({
    email: email,
    username: userId,
    password: password,
  }), {
    headers: { 'Content-Type': 'application/json' },
  });

  if (registerRes.status === 201 || registerRes.status === 200) {
    token = registerRes.json('access_token');
  } else {
    // Если пользователь уже есть — логин
    const loginRes = http.post(`${BASE_URL}/v1/auth/login`, JSON.stringify({
      login: email,
      password: password,
    }), {
      headers: { 'Content-Type': 'application/json' },
    });
    token = loginRes.json('access_token');
  }

  check(token, { 'got token': (t) => t && t.length > 0 });
  if (!token) {
    errorRate.add(1);
    return;
  }

  const authHeaders = {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json',
  };

  // 2. Search (публичный, без auth)
  const searchRes = http.get(`${BASE_URL}/v1/search?q=apothecary&limit=10`);
  check(searchRes, {
    'search status 200': (r) => r.status === 200,
    'search has results': (r) => {
      try {
        const body = r.json();
        return body.hits && body.hits.length > 0;
      } catch {
        return false;
      }
    },
  });
  if (searchRes.status !== 200) errorRate.add(1);

  sleep(0.5);

  // 3. Catalog — список аниме
  const animeListRes = http.get(`${BASE_URL}/v1/anime?limit=10&offset=0`);
  check(animeListRes, {
    'anime list status 200': (r) => r.status === 200,
  });
  if (animeListRes.status !== 200) errorRate.add(1);

  let animeId = null;
  try {
    const animeList = animeListRes.json('anime');
    if (animeList && animeList.length > 0) {
      animeId = animeList[0].id;
    }
  } catch (e) {
    errorRate.add(1);
  }

  if (!animeId) {
    errorRate.add(1);
    return;
  }

  sleep(0.3);

  // 4. Get anime details
  const animeRes = http.get(`${BASE_URL}/v1/anime/${animeId}`);
  check(animeRes, {
    'anime details status 200': (r) => r.status === 200,
  });
  if (animeRes.status !== 200) errorRate.add(1);

  sleep(0.2);

  // 5. Get episodes
  const episodesRes = http.get(`${BASE_URL}/v1/anime/${animeId}/episodes?limit=5`);
  check(episodesRes, {
    'episodes status 200': (r) => r.status === 200,
  });
  if (episodesRes.status !== 200) errorRate.add(1);

  let episodeId = null;
  try {
    const episodes = episodesRes.json('episodes');
    if (episodes && episodes.length > 0) {
      episodeId = episodes[0].id;
    }
  } catch (e) {
    errorRate.add(1);
  }

  if (!episodeId) {
    errorRate.add(1);
    return;
  }

  sleep(0.5);

  // 6. Activity — save progress
  const progressRes = http.post(`${BASE_URL}/v1/activity/progress`, JSON.stringify({
    episode_id: episodeId,
    progress_seconds: Math.floor(Math.random() * 600),
    duration_seconds: 1440,
  }), { headers: authHeaders });

  check(progressRes, {
    'activity progress status 200': (r) => r.status === 200,
  });
  if (progressRes.status !== 200) errorRate.add(1);

  sleep(0.3);

  // 7. Continue watching
  const continueRes = http.get(`${BASE_URL}/v1/activity/continue?limit=5`, { headers: authHeaders });
  check(continueRes, {
    'continue watching status 200': (r) => r.status === 200,
  });
  if (continueRes.status !== 200) errorRate.add(1);

  sleep(0.5);

  // 8. Comments — create
  const commentRes = http.post(`${BASE_URL}/v1/comments/${animeId}`, JSON.stringify({
    body: `Load test comment from ${userId}`,
  }), { headers: authHeaders });

  check(commentRes, {
    'comment create status 201 or 200': (r) => r.status === 201 || r.status === 200,
  });
  if (commentRes.status !== 201 && commentRes.status !== 200) errorRate.add(1);

  sleep(0.3);

  // 9. List comments
  const commentsListRes = http.get(`${BASE_URL}/v1/comments/${animeId}?sort=new&limit=10`);
  check(commentsListRes, {
    'comments list status 200': (r) => r.status === 200,
  });
  if (commentsListRes.status !== 200) errorRate.add(1);

  sleep(1);
}
