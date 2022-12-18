<template>
  <section class="left-sidebar">
    <nav class="left-sidebar__nav">
      <NuxtLink
        v-for="link in links"
        :key="link.href"
        :to="link.href"
        :title="link.title"
        :class="{'active': isActive(link)}"
        class="left-sidebar__link"
      >
        <span v-html="link.icon" />
      </NuxtLink>
    </nav>

    <div class="left-sidebar__info">
      <WsConnectionIcon class="left-sidebar__info-item"/>
<!--      <Logout/>-->
    </div>
  </section>
</template>

<script>
import WsConnectionIcon from "./WsConnectionIcon";
import Logout from "./Logout";

export default {
  components: {Logout, WsConnectionIcon},
  data() {
    return {
      links: [
        {
          href: '/',
          title: 'Events',
          state: (route, url) => route.path === url,
          icon: '<svg class="left-sidebar__link-icon" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path d="m19 1h-14a5.006 5.006 0 0 0 -5 5v12a5.006 5.006 0 0 0 5 5h14a5.006 5.006 0 0 0 5-5v-12a5.006 5.006 0 0 0 -5-5zm-14 2h14a3 3 0 0 1 3 3v1h-20v-1a3 3 0 0 1 3-3zm14 18h-14a3 3 0 0 1 -3-3v-9h20v9a3 3 0 0 1 -3 3zm0-8a1 1 0 0 1 -1 1h-12a1 1 0 0 1 0-2h12a1 1 0 0 1 1 1zm-4 4a1 1 0 0 1 -1 1h-8a1 1 0 0 1 0-2h8a1 1 0 0 1 1 1zm-12-12a1 1 0 1 1 1 1 1 1 0 0 1 -1-1zm3 0a1 1 0 1 1 1 1 1 1 0 0 1 -1-1zm3 0a1 1 0 1 1 1 1 1 1 0 0 1 -1-1z"/></svg>',
        },
        {
          href: '/sentry',
          title: 'Sentry logs',
          state: (route, url) => route.path.startsWith(url),
          icon: '<svg class="left-sidebar__link-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 22"><path d="M23.6 18c.5 1 .5 2 .1 2.7-.5.8-1.3 1.3-2.4 1.3h-2a19 19 0 0 0 0-2.5h1.3c.3 0 .6-.3.6-.6l-.1-.3-8.6-15.2A.6.6 0 0 0 12 3a.6.6 0 0 0-.5.3L10 5.8A18.9 18.9 0 0 1 17.4 22h-6.6a12.6 12.6 0 0 0 0-2.5 12 12 0 0 0-4-7.7l-1 1.6A10.1 10.1 0 0 1 9 22H2.7c-1 0-2-.5-2.4-1.3-.4-.8-.4-1.8.1-2.7l1.3-2.3c.7.4 1.4 1 2 1.6l-.8 1.3a.6.6 0 0 0 0 .5.6.6 0 0 0 .3.4h3.2a7.6 7.6 0 0 0-3.8-5.4l3.4-6 2 1.5c2.9 2.4 4.8 5.9 5.2 9.9H15a16.4 16.4 0 0 0-8.1-13l2.8-5C10.3.5 11.1 0 12 0c.9 0 1.7.6 2.3 1.5L23.6 18Z"/></svg>',
        },
        {
          href: '/profiler',
          title: 'Profiler',
          state: (route, url) => route.path.startsWith(url),
          icon: '<svg class="left-sidebar__link-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 25 26"><path fill-rule="evenodd" d="M11.65 0c-.8 0-1.45.64-1.45 1.44 0 .79.65 1.43 1.45 1.43h.49v.84c-.31.03-.63.08-.94.14-.52.1-.87.6-.77 1.11.1.52.6.87 1.13.77.48-.1.96-.14 1.44-.16V7a.5.5 0 0 0 1 0V5.6a9.52 9.52 0 0 1 5.86 2.58l-1.24 1.5a.5.5 0 1 0 .76.64l1.17-1.4A9.24 9.24 0 0 1 22.7 15H21a.5.5 0 0 0 0 1h1.63a9.16 9.16 0 0 1-2.42 5.1l-.84-.93a.5.5 0 0 0-.74.66l.86.97A9.48 9.48 0 0 1 14 24.06V23a.5.5 0 0 0-1 0v1.08a9.52 9.52 0 0 1-1.41-.15.97.97 0 0 0-1.14.76.96.96 0 0 0 .78 1.12 11.5 11.5 0 0 0 9.33-2.4 11.2 11.2 0 0 0 4.09-8.6 11.04 11.04 0 0 0-3.32-7.88l.8-.78.1.09.66.7.7-.67.7-.66.71-.66-.67-.7-.67-.69-.67-.69-.7.66-.7.66-.7.66.57.6-.84.83a11.4 11.4 0 0 0-5.56-2.5v-.9h.48c.8 0 1.46-.65 1.46-1.44C17 .64 16.34 0 15.54 0h-3.89Zm2.43 3.68V1.9h1.46c.27 0 .48-.21.48-.47a.48.48 0 0 0-.48-.48h-3.89a.48.48 0 0 0-.48.48c0 .26.22.47.48.47h1.46v1.74c.33 0 .65 0 .97.03Zm9.55 1.24-.67-.7-.7.67.66.69.7-.66ZM5.83 7.66c0-.27.21-.48.48-.48h1.95c.26 0 .48.21.48.48 0 .26-.22.48-.48.48H6.3a.48.48 0 0 1-.48-.48Zm-1.95 2.87c0-.27.22-.48.49-.48h3.89c.26 0 .48.21.48.48 0 .26-.22.48-.48.48H4.37a.48.48 0 0 1-.49-.48Zm-.48 8.14a.48.48 0 0 0-.49.47c0 .27.22.48.49.48h4.86c.26 0 .48-.21.48-.48a.48.48 0 0 0-.48-.47H3.4ZM0 16.27c0-.26.22-.48.49-.48h7.77c.26 0 .48.22.48.48 0 .27-.22.48-.48.48H.49a.48.48 0 0 1-.49-.48Zm2.43-3.35a.48.48 0 0 0-.49.48c0 .26.22.48.49.48h5.83c.26 0 .48-.22.48-.48a.48.48 0 0 0-.48-.48H2.43Zm3.4 9.1c0-.27.21-.48.48-.48h1.95c.26 0 .48.21.48.48 0 .26-.22.47-.48.47H6.3a.48.48 0 0 1-.48-.47Zm12.5-10.06a.95.95 0 0 0-.2-1.18.98.98 0 0 0-1.2-.08l-4.96 3.39a2.29 2.29 0 0 0-.33 3.52 2.36 2.36 0 0 0 3.69-.48l3-5.17Zm-5.26 3.7 1.44-.98-.87 1.5a.4.4 0 0 1-.62.08.39.39 0 0 1 .05-.6Z" clip-rule="evenodd"/></svg>'
        },
        {
          href: '/smtp',
          title: 'SMTP mails',
          state: (route, url) => route.path.startsWith(url),
          icon: '<svg class="left-sidebar__link-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M19,1H5A5.006,5.006,0,0,0,0,6V18a5.006,5.006,0,0,0,5,5H19a5.006,5.006,0,0,0,5-5V6A5.006,5.006,0,0,0,19,1ZM5,3H19a3,3,0,0,1,2.78,1.887l-7.658,7.659a3.007,3.007,0,0,1-4.244,0L2.22,4.887A3,3,0,0,1,5,3ZM19,21H5a3,3,0,0,1-3-3V7.5L8.464,13.96a5.007,5.007,0,0,0,7.072,0L22,7.5V18A3,3,0,0,1,19,21Z"/></svg>',
        },
        {
          href: '/inspector',
          title: 'Inspector logs',
          state: (route, url) => route.path.startsWith(url),
          icon: '<svg class="left-sidebar__link-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill-rule="evenodd" clip-rule="evenodd" d="M19 3H5a3 3 0 0 0-3 3v12a3 3 0 0 0 3 3h14a3 3 0 0 0 3-3V6a3 3 0 0 0-3-3ZM5 1a5 5 0 0 0-5 5v12a5 5 0 0 0 5 5h14a5 5 0 0 0 5-5V6a5 5 0 0 0-5-5H5Z" /><path fill-rule="evenodd" clip-rule="evenodd" d="M7 12c0-.6.4-1 1-1h10a1 1 0 1 1 0 2H8a1 1 0 0 1-1-1Z" /><path fill-rule="evenodd" clip-rule="evenodd" d="M5 8c0-.6.4-1 1-1h10a1 1 0 1 1 0 2H6a1 1 0 0 1-1-1ZM5 16c0-.6.4-1 1-1h10a1 1 0 1 1 0 2H6a1 1 0 0 1-1-1Z"/></svg>',
        },
        // {
        //   href: '/terminal',
        //   title: 'Terminal',
        //   state: (url) => this.$page.url.startsWith(url),
        //   icon: '<svg class="fleft-sidebar__link-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 22"><path fill-rule="evenodd" d="M19 2H5a3 3 0 0 0-3 3v12a3 3 0 0 0 3 3h14a3 3 0 0 0 3-3V5a3 3 0 0 0-3-3ZM5 0a5 5 0 0 0-5 5v12a5 5 0 0 0 5 5h14a5 5 0 0 0 5-5V5a5 5 0 0 0-5-5H5Z"/><path fill-rule="evenodd" d="m11.1 15 .3-.6.6-.2h5.1c.3 0 .5 0 .6.2.2.2.3.4.3.7 0 .2 0 .5-.3.6-.1.2-.3.3-.6.3H12a.8.8 0 0 1-.6-.3 1 1 0 0 1-.3-.6ZM6.3 6.4a.9.9 0 0 1 .6-.3.8.8 0 0 1 .6.3l3.4 3.6a1 1 0 0 1 .2.6 1 1 0 0 1-.2.7l-3.4 3.6-.6.3a.8.8 0 0 1-.6-.3 1 1 0 0 1-.3-.6c0-.3 0-.5.3-.7l2.8-3-2.8-3A1 1 0 0 1 6 7a1 1 0 0 1 .3-.6Z" /></svg>'
        // },
        {
          href: '/settings',
          title: 'Settings',
          state: (route, url) => route.path.startsWith(url),
          icon: '<svg class="left-sidebar__link-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill-rule="evenodd" clip-rule="evenodd" d="M19 3H5a3 3 0 0 0-3 3v12a3 3 0 0 0 3 3h14a3 3 0 0 0 3-3V6a3 3 0 0 0-3-3ZM5 1a5 5 0 0 0-5 5v12a5 5 0 0 0 5 5h14a5 5 0 0 0 5-5V6a5 5 0 0 0-5-5H5Z"/><path d="M11.6 11.8V5.4c0-.2.2-.4.4-.4s.4.2.4.4v6.4a2 2 0 0 1 0 4v2.8c0 .2-.2.4-.4.4a.4.4 0 0 1-.4-.4v-2.9a2 2 0 0 1 0-3.9ZM12 15a1.2 1.2 0 1 0 0-2.4 1.2 1.2 0 0 0 0 2.4ZM16.6 8.5V5.4c0-.2.2-.4.4-.4s.4.2.4.4v3.1a2 2 0 0 1 0 4v6.1c0 .2-.2.4-.4.4a.4.4 0 0 1-.4-.4v-6.2a2 2 0 0 1 0-3.9Zm.4 3.2a1.2 1.2 0 1 0 0-2.4 1.2 1.2 0 0 0 0 2.4ZM6.6 8.5V5.4c0-.2.2-.4.4-.4s.4.2.4.4v3.1a2 2 0 0 1 0 4v6.1c0 .2-.2.4-.4.4a.4.4 0 0 1-.4-.4v-6.2a2 2 0 0 1 0-4Zm.4 3.2a1.2 1.2 0 1 0 0-2.4 1.2 1.2 0 0 0 0 2.4Z"/></svg>'
        }
      ]
    }
  },
  methods: {
    isActive(link) {
      return link.state.bind(this)(this.$nuxt.$route, link.href)
    }
  }
}
</script>

<style lang="scss">
.left-sidebar {
  @apply bg-gray-200 dark:bg-gray-800 border-r border-gray-300 dark:border-gray-500 flex flex-col justify-between z-50;

  &__nav {
    @apply divide-y divide-gray-300 dark:divide-gray-600;
  }

  &__link {
    @apply text-blue-500 p-3 md:p-4 lg:p-5 block hover:bg-blue-500 hover:text-white relative;

    &.active {
      @apply bg-blue-700 text-blue-200;
    }

    &-icon {
      @apply fill-current;
    }
  }

  &__info {
    @apply divide-y divide-gray-300;

    &-item {
      @apply p-2 md:p-3 lg:p-4;
    }
  }

  &__supIcon {
    @apply absolute top-2 w-2 h-2 bg-red-600 right-2 rounded-full transition transition-all duration-300;
  }
}
</style>
