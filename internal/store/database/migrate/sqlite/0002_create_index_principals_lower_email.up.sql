CREATE UNIQUE INDEX index_principals_lower_email
ON principals(LOWER(principal_email));
