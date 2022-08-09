echo
echo "Setting up GitHub Package Registry"
echo "----------------------------------"
echo
echo "Follow these steps to get access to GitHub Package Registry where"
echo "Harness modules are published privately."
echo
echo "1. Go to https://github.com/settings/tokens - Sign in with your Harness Github account if needed"
echo "2. Create a token with 'repo' and 'read:packages' scopes"
read -s -p "3. Copy the token and paste it here: " githubToken
echo
echo
echo "All done. Token is saved in ~/.npmrc."

echo "@harness:registry=https://npm.pkg.github.com" > ~/.npmrc
echo "//npm.pkg.github.com/:_authToken="$githubToken >> ~/.npmrc
echo "always-auth=true" >> ~/.npmrc

echo
echo "Update yarn checksums...."
yarn --update-checksums