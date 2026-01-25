ALTER TABLE users ADD COLUMN role VARCHAR(20) DEFAULT 'user';
-- Назначаем главного админа
UPDATE users SET role = 'admin' WHERE username = 'admin';