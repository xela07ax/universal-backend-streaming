-- Пароль 'hydro-super-secret-key-2026-change-me' (bcrypt хеш)
INSERT INTO users (id, username, email, password_hash, created_at, updated_at)
VALUES(
          gen_random_uuid(),
          'admin',
          'admin@hydro.engine',
          '$2a$10$wuM1jVI4ebmjWzheO1tyP.5rGl6LxzBBg5r2v5bEk2KrLc/I3JE.a',
          NOW(),
          NOW()
      );