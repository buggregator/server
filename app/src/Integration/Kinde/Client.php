<?php

declare(strict_types=1);

namespace App\Integration\Kinde;

use GuzzleHttp\ClientInterface;
use Kinde\KindeSDK\OAuthException;
use Kinde\KindeSDK\Sdk\Enums\GrantType;
use Kinde\KindeSDK\Sdk\Enums\StorageEnums;
use Kinde\KindeSDK\Sdk\Utils\Utils;
use Nyholm\Psr7\Uri;
use Psr\Http\Message\UriInterface;

final readonly class Client
{
    private string $authorizationEndpoint;
    private string $tokenEndpoint;
    private string $logoutEndpoint;
    private JwtTokenParser $tokenParser;

    public function __construct(
        private string $domain,
        private ?string $redirectUri,
        private string $clientId,
        private string $clientSecret,
        private ?string $logoutRedirectUri,
        private SessionStorage $storage,
        private ClientInterface $client,
        private string $scopes = 'openid profile email offline',
    ) {
        if (!Utils::validationURL($this->domain)) {
            throw new \InvalidArgumentException("Please provide valid domain");
        }

        if (!Utils::validationURL($this->redirectUri)) {
            throw new \InvalidArgumentException("Please provide valid redirect_uri");
        }

        if ($this->clientSecret === '' || $this->clientSecret === '0') {
            throw new \InvalidArgumentException("Please provide client_secret");
        }

        if ($this->clientId === '' || $this->clientId === '0') {
            throw new \InvalidArgumentException("Please provide client_id");
        }

        if ($this->logoutRedirectUri === null || $this->logoutRedirectUri === '' || $this->logoutRedirectUri === '0') {
            throw new \InvalidArgumentException("Please provide logout_redirect_uri");
        }

        if (!Utils::validationURL($this->logoutRedirectUri)) {
            throw new \InvalidArgumentException("Please provide valid logout_redirect_uri");
        }

        // Other endpoints
        $this->authorizationEndpoint = $this->domain . '/oauth2/auth';
        $this->tokenEndpoint = $this->domain . '/oauth2/token';
        $this->logoutEndpoint = $this->domain . '/logout';

        $this->tokenParser = new JwtTokenParser($this->domain . '/.well-known/jwks.json');
    }

    public function getLoginUrl(): UriInterface
    {
        $this->cleanStorage();

        $state = Utils::randomString();
        $this->storage->setState($state);

        $searchParams = [
            'client_id' => $this->clientId,
            'grant_type' => GrantType::authorizationCode,
            'redirect_uri' => $this->redirectUri,
            'response_type' => 'code',
            'scope' => $this->scopes,
            'state' => $state,
            'start_page' => 'login',
        ];

        return new Uri($this->authorizationEndpoint . '?' . \http_build_query($searchParams));
    }

    public function getLogoutUrl(): UriInterface
    {
        $searchParams = [
            'redirect' => $this->logoutRedirectUri,
        ];

        return new Uri($this->logoutEndpoint . '?' . \http_build_query($searchParams));
    }

    public function cleanStorage(): void
    {
        $this->storage->clear();
    }

    public function getToken(array $queryParams): object|array
    {
        if ($this->isAuthenticated()) {
            $token = $this->storage->getToken(false);
            if (!empty($token)) {
                return $token;
            }
        }

        $formParams = [
            'client_id' => $this->clientId,
            'client_secret' => $this->clientSecret,
            'grant_type' => 'authorization_code',
            'redirect_uri' => $this->redirectUri,
            'response_type' => 'code',
        ];

        $stateServer = $queryParams['state'] ?? null;

        $this->checkStateAuthentication($stateServer);

        $error = $queryParams['error'] ?? '';
        if (!empty($error)) {
            $errorDescription = $queryParams['error_description'] ?? '';
            $msg = empty($errorDescription) ? $error : $errorDescription;
            throw new OAuthException($msg);
        }

        $authorizationCode = $queryParams['code'] ?? '';
        if (empty($authorizationCode)) {
            throw new \InvalidArgumentException('Not found code param');
        }

        $formParams['code'] = $authorizationCode;
        $codeVerifier = $this->storage->getCodeVerifier();

        if ($codeVerifier !== null && $codeVerifier !== '' && $codeVerifier !== '0') {
            $formParams['code_verifier'] = $codeVerifier;
        }

        return $this->fetchToken($formParams);
    }

    public function getUserDetails(): array
    {
        $payload = $this->tokenParser->parse(
            $this->storage->getIdToken(),
        );

        return [
            'id' => $payload['sub'] ?? '',
            'given_name' => $payload['given_name'] ?? '',
            'family_name' => $payload['family_name'] ?? '',
            'email' => $payload['email'] ?? '',
            'picture' => $payload['picture'] ?? '',
        ];
    }

    private function fetchToken(array $formParams): object
    {
        $response = $this->client->request('POST', $this->tokenEndpoint, [
            'form_params' => $formParams,
            'headers' => [
                'Kinde-SDK' => 'PHP/1.2', // current SDK version
            ],
        ]);

        $token = $response->getBody()->getContents();
        $this->storage->setToken($token);
        $tokenDecode = \json_decode($token, false);

        // Cleaning
        $this->storage->removeItem(StorageEnums::CODE_VERIFIER);
        $this->storage->removeItem(StorageEnums::STATE);

        return $tokenDecode;
    }

    public function isAuthenticated(): bool
    {
        $accessToken = $this->tokenParser->parse($this->storage->getAccessToken() ?? '');
        $timeExpired = $accessToken === null || $accessToken === [] ? 0 : $accessToken['exp'] ?? 0;
        $authenticated = $timeExpired > \time();

        if ($authenticated) {
            return true;
        }

        // Using refresh token to get new access token
        try {
            $refreshToken = $this->storage->getRefreshToken();
            if ($refreshToken !== null && $refreshToken !== '' && $refreshToken !== '0') {
                $formParams = [
                    'client_id' => $this->clientId,
                    'client_secret' => $this->clientSecret,
                    'grant_type' => 'refresh_token',
                    'refresh_token' => $refreshToken,
                ];

                $token = $this->fetchToken($formParams);
                if ($token->expires_in > 0) {
                    return true;
                }
            }
        } catch (\Throwable) {
        }

        return false;
    }

    private function checkStateAuthentication(string $stateServer): void
    {
        $storageOAuthState = $this->storage->getState();

        if ($storageOAuthState === null || $storageOAuthState === '' || $storageOAuthState === '0' || $stateServer !== $storageOAuthState) {
            throw new OAuthException("Authentication failed because it tries to validate state");
        }
    }
}
