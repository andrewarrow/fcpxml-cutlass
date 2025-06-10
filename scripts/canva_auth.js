import crypto from "crypto";
const codeVerifier = crypto.randomBytes(96).toString("base64url");
const codeChallenge = crypto.createHash("sha256").update(codeVerifier).digest("base64url");
console.log(codeVerifier)
console.log(codeChallenge)
