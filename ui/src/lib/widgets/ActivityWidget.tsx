import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { ActivitySVGURI, ClearActivity, GetActivity } from "../api/activity"
import IconImage from "../ui/icons/IconImage"
import IconExternalLink from "../ui/icons/IconExternalLink"
import IconRefersh from "../ui/icons/IconRefresh"
import IconEraser from "../ui/icons/IconEraser"
import { useAuth } from "../AuthProvider"

export default function ActivityWidget() {
    const { token } = useAuth()
    const queryClient = useQueryClient()
    const { data, refetch, isPending } = useQuery({
        queryKey: ["activity"],
        queryFn: GetActivity,
    })

    const clearMutation = useMutation({
        mutationKey: ["activity", "clear"],
        mutationFn: ClearActivity,
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["activity"] })
        },
    })

    return (
        <article className="text-indigo-950 font-manrope">
            {data && (
                <img
                    className="w-full rounded-xl shadow-md"
                    src={`${ActivitySVGURI}?id=${data.Id}`}
                    alt="activity thumbanail"
                />
            )}
            {!data && (
                <img
                    className="w-full rounded-xl shadow-md"
                    src={`https://i.pinimg.com/736x/1b/0f/cb/1b0fcbd7e77a195fad59539af48d20bb.jpg`}
                    alt="activity thumbanail"
                />
            )}

            <div className="flex mt-2 flex-row gap-2 h-12 *:shadow-md ">
                <p className="grow h-full bg-indigo-50 rounded-xl gap-2 flex flex-row items-center px-4 truncate">
                    {data && (
                        <>
                            {data.Title}
                            <span className="size-2 aspect-square bg-indigo-500 rounded-full"></span>
                            {data.Author}
                        </>
                    )}
                    {isPending && <>Loading...</>}
                </p>
            </div>
            <div className="flex mt-2 flex-row gap-2 h-12 *:shadow-md ">
                <button
                    onClick={() => refetch()}
                    className="h-full aspect-square bg-indigo-50  hover:text-pink-500 rounded-xl transition-colors cursor-pointer flex items-center justify-center"
                >
                    <IconRefersh className="size-10" />
                </button>
                <button
                    onClick={() => clearMutation.mutate(token!)}
                    className="h-full bg-indigo-50 hover:text-pink-500 rounded-xl transition-colors cursor-pointer flex items-center justify-center px-2 gap-1"
                >
                    <IconEraser className="size-8" />
                    Clear
                </button>
                <span className="grow !shadow-none"></span>
                {data && (
                    <>
                        <a
                            href={data.ThumbnailUrl}
                            target="_blank"
                            className="h-full aspect-square bg-indigo-50 hover:text-pink-500 rounded-xl transition-colors cursor-pointer flex items-center justify-center"
                        >
                            <IconImage className="size-8" />
                        </a>
                        <a
                            href={data.Url}
                            target="_blank"
                            className="h-full aspect-square bg-indigo-50 hover:text-pink-500 rounded-xl transition-colors cursor-pointer flex items-center justify-center"
                        >
                            <IconExternalLink className="size-8" />
                        </a>
                    </>
                )}
            </div>
        </article>
    )
}
