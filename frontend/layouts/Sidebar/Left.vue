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
          href: '/smtp',
          title: 'SMTP mails',
          state: (route, url) => route.path.startsWith(url),
          icon: '<svg class="left-sidebar__link-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M19,1H5A5.006,5.006,0,0,0,0,6V18a5.006,5.006,0,0,0,5,5H19a5.006,5.006,0,0,0,5-5V6A5.006,5.006,0,0,0,19,1ZM5,3H19a3,3,0,0,1,2.78,1.887l-7.658,7.659a3.007,3.007,0,0,1-4.244,0L2.22,4.887A3,3,0,0,1,5,3ZM19,21H5a3,3,0,0,1-3-3V7.5L8.464,13.96a5.007,5.007,0,0,0,7.072,0L22,7.5V18A3,3,0,0,1,19,21Z"/></svg>',
        },
        {
          href: '/sentry',
          title: 'Sentry logs',
          state: (route, url) => route.path.startsWith(url),
          icon: '<svg class="left-sidebar__link-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 22"><path d="M23.6 18c.5 1 .5 2 .1 2.7-.5.8-1.3 1.3-2.4 1.3h-2a19 19 0 0 0 0-2.5h1.3c.3 0 .6-.3.6-.6l-.1-.3-8.6-15.2A.6.6 0 0 0 12 3a.6.6 0 0 0-.5.3L10 5.8A18.9 18.9 0 0 1 17.4 22h-6.6a12.6 12.6 0 0 0 0-2.5 12 12 0 0 0-4-7.7l-1 1.6A10.1 10.1 0 0 1 9 22H2.7c-1 0-2-.5-2.4-1.3-.4-.8-.4-1.8.1-2.7l1.3-2.3c.7.4 1.4 1 2 1.6l-.8 1.3a.6.6 0 0 0 0 .5.6.6 0 0 0 .3.4h3.2a7.6 7.6 0 0 0-3.8-5.4l3.4-6 2 1.5c2.9 2.4 4.8 5.9 5.2 9.9H15a16.4 16.4 0 0 0-8.1-13l2.8-5C10.3.5 11.1 0 12 0c.9 0 1.7.6 2.3 1.5L23.6 18Z"/></svg>',
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
