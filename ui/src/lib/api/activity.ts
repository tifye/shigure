import { BASE_URL } from "./auth"

export const ActivitySVGURI = `${BASE_URL}/activity/svg`

export type ActivityData = {
    Title: string
    Author: string
    ThumbnailUrl: string
    Url: string
}

export default async function GetActivity(): Promise<ActivityData> {
    const res = await fetch(`${BASE_URL}/activity`)
    if (!res.ok) {
        throw new Error(await res.text())
    }
    return res.json();
}
