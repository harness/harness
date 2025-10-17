-- primary key is required by some database tools and dependencies.
ALTER TABLE migrations ADD PRIMARY KEY (version);