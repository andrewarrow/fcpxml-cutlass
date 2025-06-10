import crypto from "crypto";

const codeVerifier = crypto.randomBytes(96).toString("base64url");
const codeChallenge = crypto.createHash("sha256").update(codeVerifier).digest("base64url");

const clientId = process.env.CANVA_CLIENT_ID;
const authUrl = `https://www.canva.com/api/oauth/authorize?code_challenge_method=s256&response_type=code&client_id=${clientId}&redirect_uri=http%3A%2F%2F127.0.0.1%3A8080&scope=comment:write%20design:meta:read%20folder:read%20folder:write%20folder:permission:write%20design:content:write%20comment:read%20app:read%20asset:read%20brandtemplate:content:read%20design:permission:write%20asset:write%20folder:permission:read%20app:write%20brandtemplate:meta:read%20profile:read%20design:content:read%20design:permission:read&code_challenge=${codeChallenge}`;

console.log("Code Verifier:", codeVerifier);
console.log("Code Challenge:", codeChallenge);
console.log("Authorization URL:", authUrl);
