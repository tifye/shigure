import IconSidebar from "./ui/icons/IconSidebar"
import { Sidebar, useSidebar } from "./ui/sidebar"

export default function AppSidebar() {
    const { toggle } = useSidebar()

    return (
        <Sidebar>
            <button
                onClick={toggle}
                className="absolute bottom-0 right-0 group-data-[collapsed=true]:hidden md:hidden"
            >
                <IconSidebar className="size-8" />
            </button>
        </Sidebar>
    )
}
