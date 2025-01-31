import { useQuery } from "@tanstack/react-query"
import GetActivity, { ActivitySVGURI } from "../api/activity"
import IconImage from "../ui/icons/IconImage"
import IconExternalLink from "../ui/icons/IconExternalLink"
import IconRefersh from "../ui/icons/IconRefresh"
import { useMemo } from "react"

export default function ActivityWidget() {
    const { data, refetch, isPending } = useQuery({
        queryKey: ["activity"],
        queryFn: GetActivity,
    })

    return (
        <article className="text-indigo-950 font-manrope">
            <img
                className="w-full rounded-xl shadow-md"
                src={`${ActivitySVGURI}?id=${data?.ThumbnailUrl}`}
                alt="activity thumbanail"
            />

            <div className="flex mt-2 flex-row gap-2 h-12 *:shadow:md ">
                <button
                    onClick={() => refetch()}
                    className="h-full aspect-square bg-indigo-50 hover:text-pink-500 rounded-xl transition-colors cursor-pointer flex items-center justify-center"
                >
                    <IconRefersh className="size-10" />
                </button>
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
