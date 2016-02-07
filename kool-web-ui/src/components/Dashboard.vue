<template>
  <h1>Dashboard</h1>

  <div id="text">
  Something something about Kool Monkey.
  </div>
</template>

<script>
  import auth from '../auth'

  export default {
    route: {
      canActivate () {
        return auth.checkAuth()
      }
    }
  }
</script>
