import { createFileRoute } from "@tanstack/react-router"
import { ActivitySVGURI } from "../../lib/api/activity"
import ActivityWidget from "../../lib/widgets/ActivityWidget"

export const Route = createFileRoute("/_auth/")({
    component: RouteComponent,
})

function RouteComponent() {
    return (
        <div className="grid-cols-1 md:grid-cols-3 lg:grid-cols-4 grid">
            <section>
                <ActivityWidget />
            </section>
        </div>
    )
}
