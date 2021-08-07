This is a table for [The Hundred] based on an idea [Henry Moeran]
mentioned on [The Final Word cricket podcast]: as the tournament has
combined organisations for the men's and women's teams, why not make a
combined table and have the top teams in that qualify for the finals?

[The Hundred]: https://www.thehundred.com/home
[Henry Moeran]: https://twitter.com/henrymoeranBBC
[The Final Word cricket podcast]: https://finalwordcricket.com/2021/07/30/season-10-ep-14-betting-the-house-on-the-hundred/

Resources:

* Cricinfo's [women's] and [men's] tables.
  * I scrape these to construct the combined table.
* [The Hundred's API](https://cricketapi-ecb.pulselive.com/fixtures?page=0&pageSize=80&sortParameter=START_DATE&sort=asc&tournamentIds=23276%2C23274)
   * I found this on the official website but it declares all matches as
     120 balls, so I wasn't sure how to calculate net run rate
     accurately.
   * Using this would allow implementing all of the tie-breakers.
* [The Hundred's Playing Conditions](https://resources.ecb.co.uk/ecb/document/2021/05/18/562e9a6f-4608-4e79-a751-f2d706ea0de2/Playing-Conditions-Hundred-180521.pdf)
  * This has details of the various tie-breakers for the league stage.

[women's]: https://www.espncricinfo.com/series/the-hundred-women-s-competition-2021-1252659/points-table-standings
[men's]: https://www.espncricinfo.com/series/the-hundred-men-s-competition-2021-1252040/points-table-standings
