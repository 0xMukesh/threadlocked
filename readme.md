# threadlocked

a dead simple [webring](https://en.wikipedia.org/wiki/Webring) api server for threadlocked

## how to join the webring?

add your entry to [`webring.json`](./webring.json) and open a pr:

```json
{
  "name": "your-name",
  "url": "https://your-site.com"
}
```

the position of your entry in the array determines where you sit in the ring order. append at the end or insert between existing members - up to you! once your entry gets added to the webring, you can use `https://<name>.seggs.lol` as a redirect to your site.

## how to add prev / next / random to your site?

the base url for the server is `https://ring.seggs.lol`. to add previous/next/random buttons to your site, use the following html snippet:

```html
<!-- replace "your-name" with the exact name from webring.json -->
<a href="https://ring.seggs.lol/redirect?from=your-name&dir=prev">
  ← previous site
</a>

<a href="https://ring.seggs.lol/random">
  random site
</a>

<a href="https://ring.seggs.lol/redirect?from=your-name&dir=next">
  next site →
</a>
```
