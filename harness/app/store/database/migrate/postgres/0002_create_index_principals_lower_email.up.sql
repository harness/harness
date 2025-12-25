CREATE UNIQUE INDEX principals_lower_email
ON principals(LOWER(principal_email));
