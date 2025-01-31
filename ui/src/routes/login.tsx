import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { ChangeEvent, useState } from "react"
import { useAuth } from "../lib/AuthProvider"

export const Route = createFileRoute("/login")({
    component: RouteComponent,
})

function RouteComponent() {
    const [code, setCode] = useState("")
    const { login, mutation: loginMutation, isAuthenticated } = useAuth()
    const navi = useNavigate()

    if (isAuthenticated) {
        navi({ to: "/" })
        return
    }

    function onChange(ev: ChangeEvent<HTMLInputElement>) {
        const input = ev.target.value
        if (input.length > 6) return

        setCode(input)

        if (input.length < 6) return

        login(input)
    }
    return (
        <div className="top-1/2 left-1/2 -translate-x-1/2 w-48 absolute tracking-widest group">
            <input
                className="bg-white h-12 w-full rounded-md shadow-md border-0 text-center focus-within:border-2 group-focus-within:border-pink-500 border-indigo-500 focus-within:outline-0 text-xl"
                type="text"
                inputMode="numeric"
                name="passcode"
                id="passcode"
                pattern="[0-9]+"
                value={code}
                onChange={onChange}
            />
            {loginMutation.isError && (
                <div className="w-full rounded-full h-2 mt-2 bg-rose-500"></div>
            )}
            {!loginMutation.isError && (
                <div className="transition-transform w-full rounded-full h-2 mt-2 bg-indigo-500 group-focus-within:bg-pink-500 group-focus-within:scale-x-105"></div>
            )}
        </div>
    )
}
