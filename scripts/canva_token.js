import { URL } from 'url';

// Parse the callback URL from command line argument
const callbackUrl = process.argv[2];
if (!callbackUrl) {
    console.error("Usage: node canva_token.js <callback_url>");
    process.exit(1);
}

// Extract the authorization code from the URL
const url = new URL(callbackUrl);
const code = url.searchParams.get('code');

if (!code) {
    console.error("No authorization code found in the URL");
    process.exit(1);
}

console.log("Authorization code:", code);

// Get environment variables
const clientId = process.env.CANVA_CLIENT_ID;
const clientSecret = process.env.CANVA_CLIENT_SECRET;
const codeVerifier = process.env.CANVA_CODE_VERIFIER;

if (!clientId || !clientSecret || !codeVerifier) {
    console.error("Missing required environment variables: CANVA_CLIENT_ID, CANVA_CLIENT_SECRET, CANVA_CODE_VERIFIER");
    process.exit(1);
}

// Create Basic Auth header
const credentials = Buffer.from(`${clientId}:${clientSecret}`).toString('base64');

// Exchange code for token
const tokenData = new URLSearchParams({
    grant_type: 'authorization_code',
    code: code,
    code_verifier: codeVerifier,
    redirect_uri: 'http://127.0.0.1:8080'
});

fetch('https://api.canva.com/rest/v1/oauth/token', {
    method: 'POST',
    headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
        'Authorization': `Basic ${credentials}`
    },
    body: tokenData
})
.then(response => response.json())
.then(data => {
    console.log('Token response:', JSON.stringify(data, null, 2));
})
.catch(error => {
    console.error('Error:', error);
});