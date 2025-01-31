/// <reference types="vite/types/importMeta.d.ts" />
export const BASE_URL = import.meta.env.VITE_BACKEND_HOST as string

export function withAuth(token: string, init: RequestInit): RequestInit {
    const bearer = `Bearer ${token}`
    init.headers = {
        ...init.headers,
        Authorization: bearer
    }
    return init
}

export function GetWithAuth(token: string, init?: RequestInit): RequestInit {
    if (!init) {
        init = {}
    }

    init.method = "GET"
    return withAuth(token, init)
}

export function PostWithAuth(token: string, init?: RequestInit): RequestInit {
    if (!init) {
        init = {}
    }

    init.method = "POST"
    return withAuth(token, init)
}

export async function getToken(passcode: string): Promise<string> {
    const url = `${BASE_URL}/auth/token`
    const res = await fetch(url, {
        headers: {
            Passcode: passcode,
        },
    })

    const body = await res.text()
    if (!res.ok || body === "") {
        throw new Error(body)
    }

    return body
}

export async function verify(token: string): Promise<void> {
    const url = `${BASE_URL}/auth/token/verify`
    const res = await fetch(url, PostWithAuth(token))
    if (!res.ok) {
        throw new Error(await res.text())
    }
}