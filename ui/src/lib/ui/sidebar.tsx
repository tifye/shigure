import {
    ComponentProps,
    createContext,
    ForwardedRef,
    forwardRef,
    JSX,
    useCallback,
    useContext,
    useMemo,
} from "react"
import { useState } from "react"
import { cn } from "../cn"

type SidebarContext = {
    isOpen: boolean
    setOpen: (open: boolean) => void
    toggle: () => void
}

const SidebarContext = createContext<SidebarContext | null>(null)

export function useSidebar() {
    const context = useContext(SidebarContext)
    if (!context) {
        throw new Error("useSidebar must be inside a SidebarProvider")
    }
    return context
}

type SidebarProviderProps = ComponentProps<"div"> & {
    defaultOpen?: boolean
    onOpenChange?: (open: boolean) => void
}

export const SidebarProvider = forwardRef<HTMLDivElement, SidebarProviderProps>(
    SidebarRender,
)

function SidebarRender(
    {
        defaultOpen = true,
        children,
        className,
        ...props
    }: Omit<SidebarProviderProps, "ref">,
    ref: ForwardedRef<HTMLDivElement>,
): JSX.Element {
    const [open, setOpen] = useState(defaultOpen)

    const toggle = useCallback(() => {
        setOpen((old) => !old)
    }, [setOpen])

    const context = useMemo<SidebarContext>(
        () => ({
            isOpen: open,
            setOpen: setOpen,
            toggle: toggle,
        }),
        [toggle, setOpen, open],
    )
    return (
        <SidebarContext.Provider value={context}>
            <div
                data-collapsed={!open}
                className={cn(
                    "relative min-h-svh gap-4 data-[collapsed=true]:gap-0 flex flex-row items-stretch bg-indigo-500 p-4 group",
                    className,
                )}
                ref={ref}
                {...props}
            >
                {children}
            </div>
        </SidebarContext.Provider>
    )
}

export function Sidebar({
    className,
    children,
    ...props
}: ComponentProps<"div">): JSX.Element {
    return (
        <div
            className={cn(
                "relative w-56 group-data-[collapsed=true]:w-0 transition-[width]  self-stretch",
                className,
            )}
            {...props}
        >
            <div className="sticky h-full top-0 bottom-0 transition-all w-56 group-data-[collapsed=true]:-translate-x-[100%] left-0 rounded-2xl text-4xl p-4 text-indigo-100 font-konigsberg ">
                {children}
            </div>
        </div>
    )
}

export function SidebarInset({
    className,
    children,
    ...props
}: ComponentProps<"div">): JSX.Element {
    return (
        <div
            className={cn(
                "relative grow bg-stone-300 rounded-2xl p-12 overflow-x-clip",
                className,
            )}
            {...props}
        >
            {children}
        </div>
    )
}
