import {
    DefaultError,
    useMutation,
    UseMutationResult,
} from "@tanstack/react-query"
import React, { useEffect } from "react"
import { getToken, verify } from "./api/auth"
import { useNavigate } from "@tanstack/react-router"

type AuthState = {
    token: string | null
    isAuthenticated: boolean
    login: (passcode: string) => void
    isLoading: boolean
    mutation: UseMutationResult<unknown, Error, string, unknown>
    logout: () => void
}

const AuthContext = React.createContext<AuthState>({
    token: null,
    isAuthenticated: false,
    login: () => {},
    mutation: null!,
    logout: () => {},
    isLoading: false,
})

export function useAuth() {
    return React.useContext(AuthContext)
}

export default function AuthProvider({
    children,
}: {
    children: React.JSX.Element
}): React.JSX.Element {
    const [token, setToken] = React.useState<string | null>(null)
    const navi = useNavigate()

    const loginMut = useMutation<string, DefaultError, string>({
        mutationKey: ["login"],
        mutationFn: getToken,
        onSuccess(tk: string) {
            setToken(tk)
            localStorage.setItem("token", tk)

            if (window.location.pathname === "/login") {
                navi({ to: "/", from: "/login" })
            }
        },
        onError: () => {
            setToken(null)
            localStorage.removeItem("token")
        },
        retry: 0,
    })

    const verifyMut = useMutation<string, DefaultError, string>({
        mutationKey: ["verify"],
        mutationFn: async (tk: string) => {
            await verify(tk)
            return tk
        },
        onSuccess: (tk) => {
            localStorage.setItem("token", tk)
            setToken(tk)
        },
        onError: () => {
            setToken(null)
            localStorage.removeItem("token")
        },
        retry: 0,
    })

    function logout() {
        setToken(null)
        localStorage.removeItem("token")
        navi({ to: "/login" })
    }

    useEffect(() => {
        if (token !== null) {
            return
        }

        const tk = localStorage.getItem("token")
        if (tk === null) {
            setToken(null)
            return
        }

        verifyMut.mutate(tk)
    }, [])

    const authState: AuthState = {
        token: token,
        isAuthenticated:
            !loginMut.isError && !verifyMut.isError && token !== null,
        login: loginMut.mutate,
        isLoading: loginMut.isPending || verifyMut.isPending,
        mutation: loginMut,
        logout: logout,
    }
    return (
        <AuthContext.Provider value={authState}>
            {children}
        </AuthContext.Provider>
    )
}
