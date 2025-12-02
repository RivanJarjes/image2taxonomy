-- Create development database
CREATE DATABASE image2taxonomy
  WITH OWNER postgres
  ENCODING 'UTF8'
  LC_COLLATE 'C'
  LC_CTYPE 'C'
  TEMPLATE template0;

-- Create test database
CREATE DATABASE image2taxonomy_test
  WITH OWNER postgres
  ENCODING 'UTF8'
  LC_COLLATE 'C'
  LC_CTYPE 'C'
  TEMPLATE template0;

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE image2taxonomy TO postgres;
GRANT ALL PRIVILEGES ON DATABASE image2taxonomy_test TO postgres;

