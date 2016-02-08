-- Add test data
\connect monkey

-- Create an agent
INSERT INTO agent
VALUES (1, '127.0.0.1', NOW());

-- Create 2 tests
INSERT INTO test
VALUES (1, 'http://www.segundamano.mx', 30);

INSERT INTO test
VALUES (2, 'http://www.google.com', 45);

-- Enable tests for agent 1
INSERT INTO testagent
VALUES (1, 2);
