Browse files on your router or raspberry or any other embedded device through web. Lightweight analog of nextcloud/owncloud. 
- no database
- no cli configs
- only 1 json configuration file
- videos/images preview 
- share files between users. 
- planned mobile/desktop clients
- web dav server with shares feature

For configuration details see wiki
![demo](https://user-images.githubusercontent.com/9258443/66141524-7c3ed780-e60c-11e9-9171-ce92cd1e5067.gif)

ℹ INFO: Sep 2019
-added possibility to download content as .m3u playlist

ℹ INFO: June 2019

-added tls support 
-added authentication in URL, gives possibility to generate links for download without entering login and password


ℹ INFO:  Jan 2019

-modified share, now it is possible to view share like user's file(not only download), added separate menu. Possible to restrict share to the specific users, all registered users, or by external link behind DMZ.

-db and cli replaced with one single config file. When you create a new user for the first time, set "firstRun":true, this will read "open/not hashed" password, and replace it with hash.

-added possibility to auth by list of ip addresses. Change "auth.method":"ip" in order to enable. Available values:ip, proxy, none, default(login and password)

-updated build command, in order to reduce final binary size. Please rebuild with debug info before submit bug report.

-removed file type detection based on content.

-removed user commands after/before file upload ...

-removed thirdparty archive, now there is system dependency on zip tool. It will call bash and redirect zip output stream directly to the browser, bypass any buffers.

-successfully run on home router with MediaTek MT7621AT CPU

-currently preview generation limited by only images, and videos, you can disable it at all by changing "defaultUser.allowGeneratePreview".

-dependencies update: filebrowser.conf(path next to the binary, required), convert.sh(path next to the binary, only if you use previews), zip linux tool(only for downloads), and ffmpeg(only if you use previews)


ℹ INFO: Nov 25 2018,  after tries to use nextcloud as a home cloud, it was not possible to use it due to performance issues. So I've decided to adopt filebrowser project to my needs. Here it is list of things I've done:

-added thumbnail generation, it requires ffmpeg as system dependencies, limited generation only for images and videos

-added thumbnails user path at settings, and by default it must be set with "-v" short command. This path, should be <b>out</b> of the user's file scope path, otherwise it will recursively generate previews for self!

-added better file type detection at the backend

-integrated photoswipe as image slider and aplayer as audio player, and left default slider for other file types.

-added possibility to play music folder recursively by selecting required folders, and press music icon at the top. Mentioned button will play current folder without recursion. APlayer hidden by default

-deleted staticgen




fork from : https://github.com/filebrowser
