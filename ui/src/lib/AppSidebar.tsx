import { useAuth } from "./AuthProvider"
import IconSidebar from "./ui/icons/IconSidebar"
import { Sidebar, useSidebar } from "./ui/sidebar"

export default function AppSidebar() {
    const { toggle } = useSidebar()
    const { logout } = useAuth()

    return (
        <Sidebar>
            <button
                onClick={logout}
                className="bg-indigo-900 w-full h-12 rounded-xl text-lg hover:bg-indigo-800 transition-colors cursor-pointer"
            >
                Logout
            </button>
            <button
                onClick={toggle}
                className="absolute bottom-0 right-0 group-data-[collapsed=true]:hidden md:hidden"
            >
                <IconSidebar className="size-8" />
            </button>
        </Sidebar>
    )
}
