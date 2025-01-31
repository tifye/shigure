import {
    createFileRoute,
    Link,
    Outlet,
    useNavigate,
} from "@tanstack/react-router"
import { useAuth } from "../lib/AuthProvider"
import { useEffect } from "react"
import {
    SidebarProvider,
    Sidebar,
    SidebarInset,
    useSidebar,
} from "../lib/ui/sidebar"
import IconSidebar from "../lib/ui/icons/IconSidebar"
import { useIsMobile } from "../lib/use-mobile"
import AppSidebar from "../lib/AppSidebar"

export const Route = createFileRoute("/_auth")({
    component: RouteComponent,
})

function _InnerLayout() {
    const { toggle, setOpen } = useSidebar()
    // const isMobile = useIsMobile()
    // useEffect(() => {
    //     setOpen(!isMobile)
    // }, [isMobile])
    return (
        <>
            <button
                onClick={toggle}
                className="absolute m-4 top-0 left-0 text-indigo-500 hover:text-pink-500 transition-colors"
            >
                <IconSidebar className="size-8" />
            </button>
            <Outlet />
        </>
    )
}

function RouteComponent() {
    const { isAuthenticated, isLoading } = useAuth()
    return (
        <>
            {isLoading && (
                <div className="top-1/2 left-1/2 -translate-x-1/2 w-48 absolute">
                    <div className="w-full text-indigo-900 text-center">
                        Authenticating
                        <span className="bg-indigo-500 mt-2 h-2 w-full transition-all -motion-translate-y-loop-100 motion-duration-[2s] motion-ease-spring-smooth group-hover:w-1/2 mx-auto rounded-full block"></span>
                    </div>
                </div>
            )}
            {!isAuthenticated && !isLoading && (
                <div className="top-1/2 left-1/2 -translate-x-1/2 w-48 absolute">
                    <div className="w-full shadow-md  bg-indigo-500 rounded-full p-4 text-white text-center ">
                        Not authenticated
                    </div>
                    <Link to="/login" className="w-full mt-8 block group">
                        <span className="block w-full text-center">Login</span>
                        <span className="bg-green-500 mt-2 h-2 w-full transition-all group-hover:w-1/2 mx-auto rounded-full block"></span>
                    </Link>
                </div>
            )}
            {isAuthenticated && !isLoading && (
                <SidebarProvider className="bg-indigo-500" defaultOpen={false}>
                    <AppSidebar />
                    <SidebarInset>
                        <_InnerLayout />
                    </SidebarInset>
                </SidebarProvider>
            )}
        </>
    )
}
