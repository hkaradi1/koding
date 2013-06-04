neo4j = require "neo4j"
{race} = require 'sinkrow'

module.exports = class Graph
  constructor:({config})->
    @db = new neo4j.GraphDatabase(config.read + ":" + config.port);

  objectify = (incomingObjects, callback)->
    incomingObjects = [].concat(incomingObjects)
    generatedObjects = []
    for incomingObject in incomingObjects
      generatedObject = {}
      for k of incomingObject
        temp = generatedObject
        parts = k.split "."
        key = parts.pop()
        while parts.length
          part = parts.shift()
          temp = temp[part] = temp[part] or {}
        temp[key] = incomingObject[k]
      generatedObjects.push generatedObject
    callback generatedObjects

  removePrivateContent:(groupId, contents, callback)->
    query = """
      start  kd=node:koding(id={groupId})
      MATCH  kd-[:member]->users-[r:owner]-groups
      WHERE groups.name = "JGroup"
       AND ( groups.privacy = "private"
        OR  groups.visibility=  "hidden" )
      RETURN groups
      ORDER BY r.createdAtEpoch DESC
    """
    params =
      groupId   : groupId
    @db.query query, params, (err, results)=>
      if err then return callback err
      secretGroups = (result.groups.data.slug for result in results)
      filteredContent = []
      for content in contents
        filteredContent.push content if content.group not in secretGroups
      callback null, filteredContent

  fetchAll:(group, startDate, callback)->
    {groupName, groupId} = group
    start = new Date().getTime()
    # do not remove white-spaces
    query = """
        START koding=node:koding(id={groupId})
        MATCH koding-[:member]->members<-[:author]-content
        WHERE (content.name = "JTutorial"
         or content.name = "JCodeSnip"
         or content.name = "JDiscussion"
         or content.name = "JBlogPost"
         or content.name = "JStatusUpdate")
         and has(content.`meta.createdAtEpoch`)
         and content.`meta.createdAtEpoch` < {startDate}
         and content.isLowQuality! is null

      """
    if groupName isnt "koding"
      query += """
          and has(content.group)
          and content.group = "#{groupName}"
        """
    query += """
        return *
        order by content.`meta.createdAtEpoch` DESC
        limit 20
      """
    params =
      groupId   : groupId
      startDate : startDate

    @db.query query, params, (err, results)=>
      tempRes = []
      if err then callback err
      else if results.length is 0 then callback null, []
      else
        collectRelations = race (i, res, fin)=>
          id = res.id

          @fecthRelatedItems id, (err, relatedResult)=>
            if err
              callback err
              fin()
            else
              tempRes[i].relationData =  relatedResult
              fin()
        , =>
          console.log new Date().getTime() - start
          if groupName == "koding"
            @removePrivateContent  groupId, tempRes, callback
          else
            callback null, tempRes
        resultData = ( result.content.data for result in results)
        objectify resultData, (objecteds)->
          for objected in objecteds
            tempRes.push objected
            collectRelations objected

  fecthRelatedItems:(itemId, callback)->
    query = [
      'start koding=node:koding(id={itemId})'
      'match koding-[r]-all'
      'where has(koding.`meta.createdAtEpoch`)'
      'return *'
      'order by koding.`meta.createdAtEpoch` DESC'
    ].join('\n');

    params =
      itemId : itemId

    @db.query query, params, (err, results) ->
      if err then throw err
      resultData = []
      for result in results
        type = result.r.type
        data = result.all.data
        data.relationType = type
        resultData.push data

      objectify resultData, (objected)->
        respond = {}
        for obj in objected
          type = obj.relationType
          if not respond[type] then respond[type] = []
          respond[type].push obj
        callback err, respond


  fetchNewInstalledApps:(group, startDate, callback)->
    {groupId} = group

    query = [
      'START kd=node:koding(id={groupId})'
      'MATCH kd-[:member]->users<-[r:user]-koding'
      'WHERE koding.name="JApp"'
      'and r.createdAtEpoch < {startDate}'
      'return *'
      'order by r.createdAtEpoch DESC'
      'limit 20'
    ].join('\n');

    params =
      groupId   : groupId
      startDate : startDate

    @db.query query, params, (err, results) =>
      if err then throw err
      @generateInstalledApps [], results, callback

  generateInstalledApps:(resultData, results, callback)->

    if results? and results.length < 1 then return callback null, resultData
    result = results.shift()
    data = {}
    objectify result.users.data, (objected)=>
      data.user = objected
      objectify result.r.data, (objected)=>
        data.relationship = objected
        objectify result.koding.data, (objected)=>
          data.app = objected
          resultData.push data
          @generateInstalledApps resultData, results, callback

  fetchNewMembers:(group, startDate, callback)->
    {groupId} = group

    query = [
      'start  koding=node:koding(id={groupId})'
      'MATCH  koding-[r:member]->members'
      'where  members.name="JAccount"'
      'and r.createdAtEpoch < {startDate}'
      'return members'
      'order by r.createdAtEpoch DESC'
      'limit 20'
    ].join('\n');

    params =
      groupId   : groupId
      startDate : startDate

    @db.query query, params, (err, results) ->
        if err then throw err
        resultData = []
        for result in results
          data = result.members.data
          resultData.push data

        objectify resultData, (objected)->
          callback err, objected

  fetchMemberFollows:(group, startDate, callback)->
    #followers
    query = [
      'start koding=node:koding(id={groupId})'
      'MATCH koding-[:member]->followees<-[r:follower]-follower'
      'where followees.name="JAccount"'
      'and follower.name="JAccount"'
      'and r.createdAtEpoch < {startDate}'
      'return r,followees, follower'
      'order by r.createdAtEpoch DESC'
      'limit 20'
    ].join('\n');

    @fetchFollows query, group, startDate, callback

  fetchTagFollows:(group, startDate, callback)->
    #followers
    {groupName} = group
    query = [
      'start koding=node:koding(id={groupId})'
      'MATCH koding-[:member]->followees<-[r:follower]-follower'
      'where followees.name="JAccount"'
      'and follower.name="JTag"'
      'and follower.name="' + groupName + '"'
      'and r.createdAtEpoch < {startDate}'
      'return r,followees, follower'
      'order by r.createdAtEpoch DESC'
      'limit 20'
    ].join('\n');

    @fetchFollows query, group, startDate, callback

  fetchFollows:(query, group, startDate, callback)->


    {groupId} = group


    params =
      groupId   : groupId
      startDate : startDate

    @db.query query, params, (err, results)=>
      if err then throw err
      @generateFollows [], results, callback

  generateFollows:(resultData, results, callback)->

    if results? and results.length < 1 then return callback null, resultData
    result = results.shift()
    data = {}
    objectify result.follower.data, (objected)=>
      data.follower = objected
      objectify result.r.data, (objected)=>
        data.relationship = objected
        objectify result.followees.data, (objected)=>
          data.followee = objected
          resultData.push data
          @generateFollows resultData, results, callback
