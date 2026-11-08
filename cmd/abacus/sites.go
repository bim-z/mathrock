/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package algernon

import "regexp"

const (
	direct = "direct"
	page   = "page"
	api    = "api"
	graph  = "graph"
	regex  = "regex"
)

type site struct {
	slug  string
	name  string
	kind  string
	link  string
	path  string
	query string
	field string
	find  *regexp.Regexp
	skip  string
	tail  string
}

var sites = []site{
	{slug: "github", name: "GitHub", kind: direct, link: "https://github.com/{u}.png"},
	{slug: "gitlab", name: "GitLab", kind: direct, link: "https://gitlab.com/{u}.png"},
	{slug: "codeberg", name: "Codeberg", kind: direct, link: "https://codeberg.org/{u}.png"},
	{slug: "gitea", name: "Gitea", kind: direct, link: "https://gitea.com/{u}.png"},
	{slug: "tumblr", name: "Tumblr", kind: direct, link: "https://api.tumblr.com/v2/blog/{u}.tumblr.com/avatar/512"},

	{slug: "reddit", name: "Reddit", kind: api, link: "https://www.reddit.com/user/{u}/about.json", path: "data.snoovatar_img|data.icon_img"},
	{slug: "devto", name: "DEV", kind: api, link: "https://dev.to/api/users/by_username?url={q}", path: "profile_image"},
	{slug: "keybase", name: "Keybase", kind: api, link: "https://keybase.io/_/api/1.0/user/lookup.json?usernames={q}&fields=pictures", path: "them.0.pictures.primary.url"},
	{slug: "bluesky", name: "Bluesky", kind: api, link: "https://public.api.bsky.app/xrpc/app.bsky.actor.getProfile?actor={b}", path: "avatar"},
	{slug: "mastodon", name: "Mastodon", kind: api, link: "https://mastodon.social/api/v1/accounts/lookup?acct={q}", path: "avatar"},
	{slug: "chesscom", name: "Chess.com", kind: api, link: "https://api.chess.com/pub/player/{u}", path: "avatar"},
	{slug: "dockerhub", name: "Docker Hub", kind: api, link: "https://hub.docker.com/v2/users/{u}/", path: "gravatar_url"},
	{slug: "crates", name: "Crates.io", kind: api, link: "https://crates.io/api/v1/users/{u}", path: "user.avatar"},
	{slug: "huggingface", name: "Hugging Face", kind: api, link: "https://huggingface.co/api/users/{u}/overview", path: "avatarUrl"},
	{slug: "codeforces", name: "Codeforces", kind: api, link: "https://codeforces.com/api/user.info?handles={q}", path: "result.0.titlePhoto"},
	{slug: "scratch", name: "Scratch", kind: api, link: "https://api.scratch.mit.edu/users/{u}", path: "profile.images.90x90"},
	{slug: "kitsu", name: "Kitsu", kind: api, link: "https://kitsu.io/api/edge/users?filter%5Bname%5D={q}", path: "data.0.attributes.avatar.original"},
	{slug: "duolingo", name: "Duolingo", kind: api, link: "https://www.duolingo.com/2017-06-30/users?username={q}", path: "users.0.picture", tail: "/xxlarge"},

	{slug: "anilist", name: "AniList", kind: graph, link: "https://graphql.anilist.co", query: "query($name:String){User(name:$name){avatar{large}}}", field: "name", path: "data.User.avatar.large"},
	{slug: "hashnode", name: "Hashnode", kind: graph, link: "https://gql.hashnode.com", query: "query($u:String!){user(username:$u){profilePicture}}", field: "u", path: "data.user.profilePicture"},
	{slug: "leetcode", name: "LeetCode", kind: graph, link: "https://leetcode.com/graphql", query: "query($u:String!){matchedUser(username:$u){profile{userAvatar}}}", field: "u", path: "data.matchedUser.profile.userAvatar"},

	{slug: "steam", name: "Steam", kind: regex, link: "https://steamcommunity.com/id/{u}?xml=1", find: regexp.MustCompile(`(?s)<avatarFull><!\[CDATA\[(.*?)\]\]></avatarFull>`)},

	{slug: "telegram", name: "Telegram", kind: page, link: "https://t.me/{u}", skip: "t_logo"},
	{slug: "youtube", name: "YouTube", kind: page, link: "https://www.youtube.com/@{u}"},
	{slug: "soundcloud", name: "SoundCloud", kind: page, link: "https://soundcloud.com/{u}"},
	{slug: "lastfm", name: "Last.fm", kind: page, link: "https://www.last.fm/user/{u}", skip: "2a96cbd8b46e442fc41c2b86b821562f"},
	{slug: "letterboxd", name: "Letterboxd", kind: page, link: "https://letterboxd.com/{u}/"},
	{slug: "myanimelist", name: "MyAnimeList", kind: page, link: "https://myanimelist.net/profile/{u}"},
	{slug: "vimeo", name: "Vimeo", kind: page, link: "https://vimeo.com/{u}"},
	{slug: "linktree", name: "Linktree", kind: page, link: "https://linktr.ee/{u}"},
	{slug: "behance", name: "Behance", kind: page, link: "https://www.behance.net/{u}"},
	{slug: "medium", name: "Medium", kind: page, link: "https://medium.com/@{u}"},
	{slug: "substack", name: "Substack", kind: page, link: "https://substack.com/@{u}"},
	{slug: "patreon", name: "Patreon", kind: page, link: "https://www.patreon.com/{u}"},
	{slug: "kofi", name: "Ko-fi", kind: page, link: "https://ko-fi.com/{u}"},

	{slug: "instagram", name: "Instagram", kind: direct, link: "https://unavatar.io/instagram/{u}?fallback=false"},
	{slug: "x", name: "X (Twitter)", kind: direct, link: "https://unavatar.io/x/{u}?fallback=false"},
	{slug: "tiktok", name: "TikTok", kind: direct, link: "https://unavatar.io/tiktok/{u}?fallback=false"},
	{slug: "twitch", name: "Twitch", kind: direct, link: "https://unavatar.io/twitch/{u}?fallback=false"},
	{slug: "threads", name: "Threads", kind: direct, link: "https://unavatar.io/threads/{u}?fallback=false"},
	{slug: "snapchat", name: "Snapchat", kind: direct, link: "https://unavatar.io/snapchat/{u}?fallback=false"},
	{slug: "spotify", name: "Spotify", kind: direct, link: "https://unavatar.io/spotify/{u}?fallback=false"},
	{slug: "dribbble", name: "Dribbble", kind: direct, link: "https://unavatar.io/dribbble/{u}?fallback=false"},
}
